package redisclient

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestNew_EmptyURLReturnsNil(t *testing.T) {
	c, err := New(context.Background(), Config{URL: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != nil {
		t.Fatalf("expected nil client when URL empty, got %#v", c)
	}
	// Nil-receiver methods must be safe.
	if err := c.Close(); err != nil {
		t.Fatalf("Close on nil: %v", err)
	}
	if err := c.Ping(context.Background()); err == nil {
		t.Fatalf("Ping on nil should error")
	}
}

func TestNew_BadURL(t *testing.T) {
	if _, err := New(context.Background(), Config{URL: "://not-a-url"}); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestNew_ConnectAndPing(t *testing.T) {
	mr := miniredis.RunT(t)
	c, err := New(context.Background(), Config{URL: "redis://" + mr.Addr(), PoolMin: 2, PoolMax: 4})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestSetExists(t *testing.T) {
	mr := miniredis.RunT(t)
	c, err := New(context.Background(), Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()
	ctx := context.Background()

	ok, err := c.Exists(ctx, "session:jti:abc")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if ok {
		t.Fatalf("key should not exist yet")
	}

	if err := c.Set(ctx, "session:jti:abc", "revoked", 50*time.Millisecond); err != nil {
		t.Fatalf("Set: %v", err)
	}
	ok, err = c.Exists(ctx, "session:jti:abc")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Fatalf("key should exist after Set")
	}

	mr.FastForward(100 * time.Millisecond)
	ok, err = c.Exists(ctx, "session:jti:abc")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if ok {
		t.Fatalf("key should have expired")
	}
}

func TestIncrWindow_SharedCounter(t *testing.T) {
	mr := miniredis.RunT(t)
	ctx := context.Background()
	// Two independent clients simulate two app instances against shared Redis.
	a, err := New(ctx, Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	defer func() { _ = a.Close() }()
	b, err := New(ctx, Config{URL: "redis://" + mr.Addr()})
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	defer func() { _ = b.Close() }()

	key := "rate:test:user"
	if n, err := a.IncrWindow(ctx, key, time.Hour); err != nil || n != 1 {
		t.Fatalf("a incr1: n=%d err=%v", n, err)
	}
	// Instance B sees instance A's increment (AC-3).
	if n, err := b.IncrWindow(ctx, key, time.Hour); err != nil || n != 2 {
		t.Fatalf("b incr2: n=%d err=%v", n, err)
	}
	if n, err := a.IncrWindow(ctx, key, time.Hour); err != nil || n != 3 {
		t.Fatalf("a incr3: n=%d err=%v", n, err)
	}

	// Window resets after TTL elapses.
	mr.FastForward(2 * time.Hour)
	if n, err := a.IncrWindow(ctx, key, time.Hour); err != nil || n != 1 {
		t.Fatalf("after window: n=%d err=%v", n, err)
	}
}

func TestPoolHelpers(t *testing.T) {
	if got := poolMax(0); got != DefaultPoolMax {
		t.Fatalf("poolMax(0)=%d want %d", got, DefaultPoolMax)
	}
	if got := poolMax(7); got != 7 {
		t.Fatalf("poolMax(7)=%d want 7", got)
	}
	if got := poolMin(0); got != DefaultPoolMin {
		t.Fatalf("poolMin(0)=%d want %d", got, DefaultPoolMin)
	}
	if got := poolMin(3); got != 3 {
		t.Fatalf("poolMin(3)=%d want 3", got)
	}
}
