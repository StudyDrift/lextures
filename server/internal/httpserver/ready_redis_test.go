package httpserver

import (
	"context"
	"net/http"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/lextures/lextures/server/internal/redisclient"
)

// TestHealthProbe_RedisDown verifies that when Redis is configured but
// unreachable, the readiness probe fails (plan 17.2 FR-1 / 17.8 AC-2).
func TestHealthProbe_RedisDown(t *testing.T) {
	mr := miniredis.RunT(t)
	rc, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = rc.Close() })

	probe := NewHealthProbe(nil, rc, nil)
	resp, code := probe.Ready(context.Background())
	if code != http.StatusServiceUnavailable {
		t.Fatalf("code %d", code)
	}
	if resp.Checks["postgres"] != "fail" || resp.Checks["redis"] != "ok" {
		t.Fatalf("before redis down: %+v", resp)
	}

	mr.Close()
	resp, code = probe.Ready(context.Background())
	if code != http.StatusServiceUnavailable {
		t.Fatalf("code after redis down: %d", code)
	}
	if resp.Status != "unhealthy" || resp.Checks["redis"] != "fail" {
		t.Fatalf("after redis down: %+v", resp)
	}
}
