package aigateway

import (
	"testing"

	"github.com/google/uuid"
)

func TestUserIDHash_Deterministic(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	a := UserIDHash("secret", id)
	b := UserIDHash("secret", id)
	if a != b || a == "" {
		t.Fatalf("hash not stable: %q %q", a, b)
	}
}

func TestContentHash_Empty(t *testing.T) {
	h := ContentHash("")
	if len(h) != 64 {
		t.Fatalf("want sha256 hex len 64, got %d", len(h))
	}
}

func TestBlockMessage(t *testing.T) {
	if BlockMessage(BlockOptOut) == "" {
		t.Fatal("expected message")
	}
}

func TestModelAllowed(t *testing.T) {
	if !modelAllowed([]string{"anthropic/claude-3.5-sonnet"}, "anthropic/claude-3.5-sonnet") {
		t.Fatal("expected match")
	}
	if modelAllowed([]string{"other"}, "anthropic/claude-3.5-sonnet") {
		t.Fatal("expected no match")
	}
}
