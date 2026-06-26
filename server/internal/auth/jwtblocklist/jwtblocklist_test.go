package jwtblocklist

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/lextures/lextures/server/internal/redisclient"
)

func newTestBlocklist(t *testing.T) (*Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c, err := redisclient.New(context.Background(), redisclient.Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("redis: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return New(c), mr
}

func TestNew_NilClient(t *testing.T) {
	if New(nil) != nil {
		t.Fatalf("New(nil) should be nil")
	}
}

func TestRevokeAndCheck(t *testing.T) {
	bl, mr := newTestBlocklist(t)
	ctx := context.Background()

	revoked, err := bl.IsRevoked(ctx, "jti-1")
	if err != nil || revoked {
		t.Fatalf("expected not revoked, got revoked=%v err=%v", revoked, err)
	}

	if err := bl.Revoke(ctx, "jti-1", time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	revoked, err = bl.IsRevoked(ctx, "jti-1")
	if err != nil || !revoked {
		t.Fatalf("expected revoked, got revoked=%v err=%v", revoked, err)
	}

	// TTL self-expiry.
	mr.FastForward(2 * time.Minute)
	revoked, err = bl.IsRevoked(ctx, "jti-1")
	if err != nil || revoked {
		t.Fatalf("expected expired, got revoked=%v err=%v", revoked, err)
	}
}

func TestRevoke_NonPositiveTTLNoop(t *testing.T) {
	bl, _ := newTestBlocklist(t)
	ctx := context.Background()
	if err := bl.Revoke(ctx, "jti-x", 0); err != nil {
		t.Fatalf("Revoke ttl=0: %v", err)
	}
	revoked, err := bl.IsRevoked(ctx, "jti-x")
	if err != nil || revoked {
		t.Fatalf("ttl=0 should not store, got revoked=%v err=%v", revoked, err)
	}
}

func TestEmptyJTI(t *testing.T) {
	bl, _ := newTestBlocklist(t)
	ctx := context.Background()
	if err := bl.Revoke(ctx, "", time.Minute); err != nil {
		t.Fatalf("Revoke empty: %v", err)
	}
	revoked, err := bl.IsRevoked(ctx, "")
	if err != nil || revoked {
		t.Fatalf("empty jti should be not revoked, got revoked=%v err=%v", revoked, err)
	}
}
