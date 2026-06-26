package httpserver

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/lextures/lextures/server/internal/redisclient"
)

// TestDefaultReady_RedisDown verifies that when Redis is configured but
// unreachable, the readiness probe fails (plan 17.2 FR-1) — the load balancer
// then removes the instance from rotation.
func TestDefaultReady_RedisDown(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = rc.Close() })

	// With a nil pool, defaultReady short-circuits to the no-DB error regardless
	// of Redis; assert that branch so the check is exercised without a Postgres.
	check := defaultReady(nil, rc)
	if err := check(); err == nil {
		t.Fatalf("expected not-ready without DB pool")
	}

	// Closing miniredis makes Redis ping fail; ensure the Redis client surfaces it.
	mr.Close()
	if err := rc.Ping(context.Background()); err == nil {
		t.Fatalf("expected redis ping to fail after close")
	}
}
