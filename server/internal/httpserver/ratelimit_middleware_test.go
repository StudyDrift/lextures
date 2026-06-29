package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/redisclient"
)

func rlDeps(t *testing.T, rl config.RateLimits) Deps {
	t.Helper()
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = rc.Close() })
	cfg := config.Config{JWTSecret: "test-secret", RateLimits: rl}
	return Deps{Config: cfg, Redis: rc}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func enabledLimits() config.RateLimits {
	rl := config.DefaultRateLimits()
	rl.Enabled = true
	rl.Auth.Limit = 10
	rl.Global.Limit = 1000 // high so the auth limit is what trips
	return rl
}

// AC-1: 11 login requests within a minute → the 11th is 429 with Retry-After.
func TestRateLimit_AuthEndpoint429(t *testing.T) {
	d := rlDeps(t, enabledLimits())
	h := d.rateLimitMiddleware(d.buildRateLimiter())(okHandler())

	var last *httptest.ResponseRecorder
	for i := 1; i <= 11; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "198.51.100.5:40000"
		last = httptest.NewRecorder()
		h.ServeHTTP(last, req)
		if i <= 10 && last.Code != http.StatusOK {
			t.Fatalf("request %d: got %d want 200", i, last.Code)
		}
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("11th request: got %d want 429", last.Code)
	}
	if last.Header().Get("Retry-After") == "" {
		t.Fatalf("429 must include Retry-After header")
	}
	if last.Header().Get("X-RateLimit-Limit") != "10" {
		t.Fatalf("X-RateLimit-Limit=%q want 10", last.Header().Get("X-RateLimit-Limit"))
	}
	if ct := last.Header().Get("Content-Type"); ct != "application/problem+json; charset=utf-8" {
		t.Fatalf("429 content-type=%q want problem+json", ct)
	}
}

// A different IP is tracked independently.
func TestRateLimit_PerIPIsolation(t *testing.T) {
	d := rlDeps(t, enabledLimits())
	h := d.rateLimitMiddleware(d.buildRateLimiter())(okHandler())

	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "198.51.100.5:40000"
		h.ServeHTTP(httptest.NewRecorder(), req)
	}
	// Fresh IP must still be allowed.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.77:40000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("different IP: got %d want 200", rec.Code)
	}
}

// AC-5: an allowlisted IP bypasses the per-IP limit.
func TestRateLimit_AllowlistBypass(t *testing.T) {
	rl := enabledLimits()
	rl.IPAllowlist = []string{"203.0.113.0/24"}
	d := rlDeps(t, rl)
	h := d.rateLimitMiddleware(d.buildRateLimiter())(okHandler())

	for i := 0; i < 25; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "203.0.113.5:40000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("allowlisted request %d: got %d want 200", i+1, rec.Code)
		}
	}
}

// Disabled feature flag = no limiting (backward compatibility).
func TestRateLimit_DisabledPassesThrough(t *testing.T) {
	rl := enabledLimits()
	rl.Enabled = false
	d := rlDeps(t, rl)
	h := d.rateLimitMiddleware(d.buildRateLimiter())(okHandler())
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "198.51.100.5:40000"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("disabled: request %d got %d want 200", i+1, rec.Code)
		}
	}
}

// Monitor-only mode records but does not block.
func TestRateLimit_MonitorOnly(t *testing.T) {
	rl := enabledLimits()
	rl.MonitorOnly = true
	d := rlDeps(t, rl)
	h := d.rateLimitMiddleware(d.buildRateLimiter())(okHandler())
	var last *httptest.ResponseRecorder
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "198.51.100.5:40000"
		last = httptest.NewRecorder()
		h.ServeHTTP(last, req)
	}
	if last.Code != http.StatusOK {
		t.Fatalf("monitor-only must not block, got %d", last.Code)
	}
}

// Untrusted peer cannot spoof a different IP via X-Forwarded-For (security).
func TestRateLimit_SpoofedXFFIgnored(t *testing.T) {
	d := rlDeps(t, enabledLimits())
	h := d.rateLimitMiddleware(d.buildRateLimiter())(okHandler())
	var last *httptest.ResponseRecorder
	for i := 0; i < 12; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "198.51.100.5:40000"
		// Attacker rotates a forged client IP each request; the untrusted peer
		// header must be ignored so the same peer is still throttled.
		req.Header.Set("X-Forwarded-For", "1.2.3."+strconv.Itoa(i+1))
		last = httptest.NewRecorder()
		h.ServeHTTP(last, req)
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("spoofed XFF must not bypass limit, got %d", last.Code)
	}
}
