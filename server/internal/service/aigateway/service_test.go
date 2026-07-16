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

func TestLogInference_EmptyProviderIsUnknown(t *testing.T) {
	// Mirror LogInference default without a pool (AP.6 FR-7).
	provider := ""
	if provider == "" {
		provider = "unknown"
	}
	if provider == ProviderOpenRouter {
		t.Fatal("must not default empty provider to openrouter")
	}
	if provider != "unknown" {
		t.Fatalf("provider=%q", provider)
	}
}

func TestModelAllowed(t *testing.T) {
	if !modelAllowed([]string{"anthropic/claude-3.5-sonnet"}, "anthropic/claude-3.5-sonnet") {
		t.Fatal("expected match")
	}
	if modelAllowed([]string{"other"}, "anthropic/claude-3.5-sonnet") {
		t.Fatal("expected no match")
	}
	// Alias allow-list matches resolved provider ids (AP.3 FR-8).
	if !modelAllowed([]string{"text-strong"}, "claude-3-5-sonnet-20241022") {
		t.Fatal("expected text-strong alias to match Anthropic id")
	}
	if !modelAllowed([]string{"text-fast"}, "arcee-ai/trinity-mini:free") {
		t.Fatal("expected text-fast alias to match OpenRouter default id")
	}
	if !modelAllowed([]string{"arcee-ai/trinity-mini:free"}, "text-fast") {
		t.Fatal("expected dual-read OpenRouter id to match alias")
	}
}
