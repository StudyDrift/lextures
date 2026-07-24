package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestClientIPMiddleware_IgnoresForgedXFFFromUntrustedPeer(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{
		RateLimits: config.RateLimits{
			TrustedProxies: []string{"10.0.0.0/8"},
		},
	}}
	var seen string
	h := d.clientIPMiddleware()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = r.RemoteAddr
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "203.0.113.9:1234" {
		t.Fatalf("RemoteAddr = %q, want peer address", seen)
	}
}

func TestClientIPMiddleware_TrustedProxyUsesXFF(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{
		RateLimits: config.RateLimits{
			TrustedProxies: []string{"10.0.0.0/8"},
		},
	}}
	var seen string
	h := d.clientIPMiddleware()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = r.RemoteAddr
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "198.51.100.7:443" {
		t.Fatalf("RemoteAddr = %q, want client IP with original port", seen)
	}
}
