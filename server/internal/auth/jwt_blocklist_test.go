package auth

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeBlocklist is an in-memory auth.Blocklist for unit tests.
type fakeBlocklist struct {
	mu      sync.Mutex
	entries map[string]time.Time // jti -> expiry
	now     func() time.Time
	failErr error
}

func newFakeBlocklist() *fakeBlocklist {
	return &fakeBlocklist{entries: map[string]time.Time{}, now: fixedNow}
}

func (f *fakeBlocklist) IsRevoked(_ context.Context, jti string) (bool, error) {
	if f.failErr != nil {
		return false, f.failErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	exp, ok := f.entries[jti]
	if !ok {
		return false, nil
	}
	if f.now().After(exp) {
		delete(f.entries, jti)
		return false, nil
	}
	return true, nil
}

func (f *fakeBlocklist) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	if f.failErr != nil {
		return f.failErr
	}
	if ttl <= 0 {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries[jti] = f.now().Add(ttl)
	return nil
}

func TestVerify_RevokedTokenRejected(t *testing.T) {
	bl := newFakeBlocklist()
	signer := newTestSigner("unit-test-secret").WithBlocklist(bl)
	ctx := context.Background()

	tok, err := signer.Sign(ctx, userID, "a@b.com", "", "", nil)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	// Valid before revocation.
	if _, err := signer.Verify(ctx, tok); err != nil {
		t.Fatalf("Verify before revoke: %v", err)
	}
	// Revoke, then a (different) instance must reject it.
	if err := signer.RevokeToken(ctx, tok); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
	other := newTestSigner("unit-test-secret").WithBlocklist(bl)
	if _, err := other.Verify(ctx, tok); err == nil {
		t.Fatalf("expected revoked token to be rejected on second instance")
	}
}

func TestRevokeToken_NoBlocklistNoop(t *testing.T) {
	signer := newTestSigner("unit-test-secret")
	tok, _ := signer.Sign(context.Background(), userID, "a@b.com", "", "", nil)
	if err := signer.RevokeToken(context.Background(), tok); err != nil {
		t.Fatalf("RevokeToken without blocklist should be no-op: %v", err)
	}
}

func TestVerify_BlocklistErrorFailsOpen(t *testing.T) {
	bl := newFakeBlocklist()
	bl.failErr = context.DeadlineExceeded
	signer := newTestSigner("unit-test-secret").WithBlocklist(bl)
	ctx := context.Background()
	tok, _ := signer.Sign(ctx, userID, "a@b.com", "", "", nil)
	// Redis unavailable must not break auth (fail-open).
	if _, err := signer.Verify(ctx, tok); err != nil {
		t.Fatalf("Verify should fail open on blocklist error: %v", err)
	}
}

func TestRevokeToken_ExpiredTokenNoop(t *testing.T) {
	bl := newFakeBlocklist()
	signer := newTestSigner("unit-test-secret").WithBlocklist(bl)
	signer.ttl = -time.Minute // sign an already-expired token
	ctx := context.Background()
	tok, err := signer.Sign(ctx, userID, "a@b.com", "", "", nil)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if err := signer.RevokeToken(ctx, tok); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
	bl.mu.Lock()
	n := len(bl.entries)
	bl.mu.Unlock()
	if n != 0 {
		t.Fatalf("expired token should not be stored, got %d entries", n)
	}
}
