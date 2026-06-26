package httpserver

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/redisclient"
)

// TestAltTextRateLimit_SharedAcrossInstances proves the per-user limit is
// enforced from a shared Redis counter regardless of which instance serves the
// request (plan 17.2 FR-3 / AC-3).
func TestAltTextRateLimit_SharedAcrossInstances(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = rc.Close() })

	// Two Deps share the same Redis = two app instances.
	instA := Deps{Redis: rc}
	instB := Deps{Redis: rc}
	ctx := context.Background()
	uid := uuid.New()

	// Exhaust the limit across both instances combined.
	for i := 0; i < altTextRateLimitPerH; i++ {
		var ok bool
		if i%2 == 0 {
			ok = instA.checkAltTextRateLimit(ctx, uid)
		} else {
			ok = instB.checkAltTextRateLimit(ctx, uid)
		}
		if !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	// The next request on EITHER instance must be blocked.
	if instB.checkAltTextRateLimit(ctx, uid) {
		t.Fatalf("expected instance B to block after shared limit reached")
	}
	if instA.checkAltTextRateLimit(ctx, uid) {
		t.Fatalf("expected instance A to block after shared limit reached")
	}
}

// TestAltTextRateLimit_InProcessFallback verifies single-instance behaviour when
// Redis is not configured.
func TestAltTextRateLimit_InProcessFallback(t *testing.T) {
	d := Deps{} // no Redis
	ctx := context.Background()
	uid := uuid.New()
	for i := 0; i < altTextRateLimitPerH; i++ {
		if !d.checkAltTextRateLimit(ctx, uid) {
			t.Fatalf("request %d should be allowed in-process", i+1)
		}
	}
	if d.checkAltTextRateLimit(ctx, uid) {
		t.Fatalf("expected in-process limiter to block after limit")
	}
}
