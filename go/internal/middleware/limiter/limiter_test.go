package limiter

import (
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestClientIPIgnoresForwardedWhenProxyUntrusted(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/login", nil)
	r.RemoteAddr = "203.0.113.5:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")

	if got := ClientIP(r, nil); got != "203.0.113.5" {
		t.Fatalf("ClientIP = %q, mau %q", got, "203.0.113.5")
	}
}

func TestClientIPUsesRightmostUntrustedForwarded(t *testing.T) {
	trusted := ParseTrustedProxies([]string{"10.0.0.0/8"})
	r := httptest.NewRequest("GET", "/api/login", nil)
	r.RemoteAddr = "10.0.0.2:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.7, 10.0.0.1")

	if got := ClientIP(r, trusted); got != "198.51.100.7" {
		t.Fatalf("ClientIP = %q, mau %q", got, "198.51.100.7")
	}
}

func TestClientIPFallsBackToRemoteForTrustedProxy(t *testing.T) {
	trusted := ParseTrustedProxies([]string{"10.0.0.0/8"})
	r := httptest.NewRequest("GET", "/api/login", nil)
	r.RemoteAddr = "10.0.0.2:1234"

	if got := ClientIP(r, trusted); got != "10.0.0.2" {
		t.Fatalf("ClientIP = %q, mau %q", got, "10.0.0.2")
	}
}

func TestKeyedLimiterAllowsUntilBurst(t *testing.T) {
	l := NewKeyedLimiter(rate.Every(time.Hour), 2, time.Minute)

	if !l.Allow("user:1") {
		t.Fatal("request pertama seharusnya diizinkan")
	}
	if !l.Allow("user:1") {
		t.Fatal("request kedua seharusnya diizinkan")
	}
	if l.Allow("user:1") {
		t.Fatal("request ketiga seharusnya ditolak")
	}
	if !l.Allow("user:2") {
		t.Fatal("bucket user lain tidak boleh terpengaruh")
	}
}
