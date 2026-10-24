package limiter

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// KeyedLimiter membatasi request berdasarkan kunci arbitrer, mis. IP, user ID,
// atau kombinasi keduanya. Setiap kunci memiliki bucket token sendiri.
type KeyedLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
	ttl      time.Duration
}

func NewKeyedLimiter(rps rate.Limit, burst int, ttl time.Duration) *KeyedLimiter {
	l := &KeyedLimiter{
		visitors: make(map[string]*visitor),
		limit:    rps,
		burst:    burst,
		ttl:      ttl,
	}
	go l.cleanup()
	return l
}

func (l *KeyedLimiter) getVisitor(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, exists := l.visitors[key]
	if !exists {
		lim := rate.NewLimiter(l.limit, l.burst)
		l.visitors[key] = &visitor{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (l *KeyedLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		l.mu.Lock()
		for key, v := range l.visitors {
			if time.Since(v.lastSeen) > l.ttl {
				delete(l.visitors, key)
			}
		}
		l.mu.Unlock()
	}
}

func (l *KeyedLimiter) Allow(key string) bool {
	return l.getVisitor(key).Allow()
}

func (l *KeyedLimiter) retryAfter() string {
	if l.limit <= 0 {
		return "1"
	}
	seconds := int(math.Ceil(1 / float64(l.limit)))
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

// Middleware membalas 429 dengan Retry-After ketika bucket untuk kunci
// tersebut habis. keyFunc dipanggil untuk setiap request.
func (l *KeyedLimiter) Middleware(
	keyFunc func(*http.Request) string,
	message string,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := keyFunc(r)
		if key == "" {
			key = "unknown"
		}
		if !l.getVisitor(key).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", l.retryAfter())
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
			return
		}
		next(w, r)
	}
}

// IPRateLimiter adalah KeyedLimiter yang memakai IP klien sebagai kunci.
type IPRateLimiter struct {
	*KeyedLimiter
	trusted []*net.IPNet
}

func NewIPRateLimiter(
	rps rate.Limit,
	burst int,
	ttl time.Duration,
	trustedProxies []string,
) *IPRateLimiter {
	return &IPRateLimiter{
		KeyedLimiter: NewKeyedLimiter(rps, burst, ttl),
		trusted:      ParseTrustedProxies(trustedProxies),
	}
}

func (l *IPRateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return l.KeyedLimiter.Middleware(
		func(r *http.Request) string { return ClientIP(r, l.trusted) },
		"Terlalu banyak percobaan. Coba beberapa detik lagi.",
		next,
	)
}

// ParseTrustedProxies mengubah daftar IP/CIDR dari konfigurasi menjadi
// jaringan yang bisa dibandingkan. Entri tidak valid diabaikan.
func ParseTrustedProxies(raw []string) []*net.IPNet {
	var result []*net.IPNet
	for _, entry := range raw {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			result = append(result, network)
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			result = append(result, &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(bits, bits),
			})
		}
	}
	return result
}

// ClientIP mengembalikan alamat klien yang bisa dipercaya.
//
// X-Forwarded-For/X-Real-IP hanya dibaca bila koneksi langsung berasal dari
// reverse proxy yang terdaftar di TRUSTED_PROXIES. Tanpa daftar itu, header
// apa pun dari klien diabaikan sehingga penyerang tidak bisa memalsukan IP
// dan melewati rate limit.
func ClientIP(r *http.Request, trusted []*net.IPNet) string {
	remote := hostOnly(r.RemoteAddr)
	if remote == "" {
		return "unknown"
	}
	if !ipInNetworks(remote, trusted) {
		return remote
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			candidate := strings.TrimSpace(parts[i])
			if net.ParseIP(candidate) == nil {
				continue
			}
			if !ipInNetworks(candidate, trusted) {
				return candidate
			}
		}
	}

	if real := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(real) != nil {
		return real
	}

	return remote
}

func hostOnly(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err == nil {
		return host
	}
	return remote
}

func ipInNetworks(raw string, networks []*net.IPNet) bool {
	ip := net.ParseIP(hostOnly(raw))
	if ip == nil {
		return false
	}
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
