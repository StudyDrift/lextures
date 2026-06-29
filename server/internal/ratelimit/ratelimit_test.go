package ratelimit

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/lextures/lextures/server/internal/config"
)

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestAllow_SlidingWindowBlocksOverLimit(t *testing.T) {
	l := New(testRedis(t), "secret", config.RateLimits{})
	rule := config.RateLimitRule{Limit: 3, Window: time.Minute}
	ctx := context.Background()
	key := l.IPKey("198.51.100.7", "auth")

	for i := 1; i <= 3; i++ {
		d := l.Allow(ctx, key, rule, LimitTypeAuth)
		if !d.Allowed {
			t.Fatalf("request %d should be allowed", i)
		}
		if d.Remaining != 3-i {
			t.Fatalf("request %d remaining=%d want %d", i, d.Remaining, 3-i)
		}
	}
	d := l.Allow(ctx, key, rule, LimitTypeAuth)
	if d.Allowed {
		t.Fatalf("4th request must be blocked")
	}
	if d.RetryAfter < 1 {
		t.Fatalf("blocked request must have RetryAfter >= 1, got %d", d.RetryAfter)
	}
	if d.Remaining != 0 {
		t.Fatalf("blocked Remaining=%d want 0", d.Remaining)
	}
}

func TestAllow_SharedAcrossInstances(t *testing.T) {
	rdb := testRedis(t)
	// Two limiters, one Redis = two app instances (AC-3).
	a := New(rdb, "secret", config.RateLimits{})
	b := New(rdb, "secret", config.RateLimits{})
	rule := config.RateLimitRule{Limit: 4, Window: time.Minute}
	ctx := context.Background()
	const ip = "203.0.113.9"

	for i := 0; i < 4; i++ {
		var d Decision
		if i%2 == 0 {
			d = a.Allow(ctx, a.IPKey(ip, "global"), rule, LimitTypeGlobal)
		} else {
			d = b.Allow(ctx, b.IPKey(ip, "global"), rule, LimitTypeGlobal)
		}
		if !d.Allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if a.Allow(ctx, a.IPKey(ip, "global"), rule, LimitTypeGlobal).Allowed {
		t.Fatalf("instance A must block after shared limit reached")
	}
	if b.Allow(ctx, b.IPKey(ip, "global"), rule, LimitTypeGlobal).Allowed {
		t.Fatalf("instance B must block after shared limit reached")
	}
}

func TestAllow_FailsOpenWithoutRedis(t *testing.T) {
	l := New(nil, "secret", config.RateLimits{})
	rule := config.RateLimitRule{Limit: 1, Window: time.Minute}
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		d := l.Allow(ctx, "rl:ip:x:global", rule, LimitTypeGlobal)
		if !d.Allowed {
			t.Fatalf("fail-open: request %d should be allowed", i+1)
		}
		if d.Limit != 1 {
			t.Fatalf("fail-open Limit=%d want 1", d.Limit)
		}
	}
}

func TestAllow_WindowExpiry(t *testing.T) {
	l := New(testRedis(t), "secret", config.RateLimits{})
	rule := config.RateLimitRule{Limit: 1, Window: 50 * time.Millisecond}
	ctx := context.Background()
	key := l.IPKey("192.0.2.1", "auth")

	if !l.Allow(ctx, key, rule, LimitTypeAuth).Allowed {
		t.Fatalf("first request should be allowed")
	}
	if l.Allow(ctx, key, rule, LimitTypeAuth).Allowed {
		t.Fatalf("second request within window should be blocked")
	}
	time.Sleep(60 * time.Millisecond)
	if !l.Allow(ctx, key, rule, LimitTypeAuth).Allowed {
		t.Fatalf("request after window should be allowed again")
	}
}

func TestAllowlisted(t *testing.T) {
	l := New(nil, "secret", config.RateLimits{IPAllowlist: []string{"203.0.113.0/24"}})
	if !l.Allowlisted("203.0.113.5") {
		t.Fatalf("203.0.113.5 should be allowlisted")
	}
	if l.Allowlisted("198.51.100.1") {
		t.Fatalf("198.51.100.1 should not be allowlisted")
	}
}

func TestClientIP_UntrustedPeerCannotSpoof(t *testing.T) {
	trusted := ParseCIDRs([]string{"10.0.0.0/8"})
	hdr := http.Header{}
	hdr.Set("X-Forwarded-For", "1.2.3.4")
	// Peer is NOT a trusted proxy: the forged header must be ignored.
	got := ClientIP("198.51.100.99:5555", hdr, trusted)
	if got != "198.51.100.99" {
		t.Fatalf("untrusted peer: got %q want raw peer 198.51.100.99", got)
	}
}

func TestClientIP_TrustedProxyForwardsClient(t *testing.T) {
	trusted := ParseCIDRs([]string{"10.0.0.0/8"})
	hdr := http.Header{}
	hdr.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.2")
	got := ClientIP("10.0.0.2:5555", hdr, trusted)
	if got != "203.0.113.7" {
		t.Fatalf("trusted proxy: got %q want client 203.0.113.7", got)
	}
}

func TestClientIP_TrustedProxyRealIP(t *testing.T) {
	trusted := ParseCIDRs([]string{"10.0.0.0/8"})
	hdr := http.Header{}
	hdr.Set("X-Real-IP", "203.0.113.42")
	got := ClientIP("10.0.0.2:80", hdr, trusted)
	if got != "203.0.113.42" {
		t.Fatalf("X-Real-IP from trusted proxy: got %q", got)
	}
}

func TestIPKey_HashedAndStable(t *testing.T) {
	l := New(nil, "secret", config.RateLimits{})
	k1 := l.IPKey("198.51.100.7", "auth")
	k2 := l.IPKey("198.51.100.7", "auth")
	if k1 != k2 {
		t.Fatalf("key must be stable: %q vs %q", k1, k2)
	}
	if k1 == "rl:ip:198.51.100.7:auth" {
		t.Fatalf("IP must be hashed, not stored raw: %q", k1)
	}
}
