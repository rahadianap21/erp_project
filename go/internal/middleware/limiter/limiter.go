package limiter

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
	ttl      time.Duration
}

func NewIPRateLimiter(rps rate.Limit, burst int, ttl time.Duration) *IPRateLimiter {
	l := &IPRateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rps,
		burst:    burst,
		ttl:      ttl,
	}
	go l.cleanup()
	return l
}

func (l *IPRateLimiter) getVisitor(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, exists := l.visitors[ip]
	if !exists {
		lim := rate.NewLimiter(l.limit, l.burst)
		l.visitors[ip] = &visitor{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (l *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		l.mu.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > l.ttl {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *IPRateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		} else if real := r.Header.Get("X-Real-IP"); real != "" {
			ip = real
		}

		lim := l.getVisitor(ip)
		if !lim.Allow() {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"Terlalu banyak percobaan login. Coba beberapa detik lagi."}`))
			return
		}
		next(w, r)
	}
}
