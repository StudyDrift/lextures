package apitokens

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashClientIP(t *testing.T) {
	t.Parallel()
	a := HashClientIP("secret-key", "203.0.113.10")
	b := HashClientIP("secret-key", "203.0.113.10")
	if a == "" || a != b {
		t.Fatalf("hash mismatch: %q %q", a, b)
	}
	if HashClientIP("", "1.2.3.4") != "" {
		t.Fatal("expected empty without key")
	}
}

func TestRecordAndFlushUsage(t *testing.T) {
	t.Parallel()
	ResetUsageQueue()
	id := uuid.New()
	RecordUsage(id, "abc123")
	ResetUsageQueue()
}

func TestRotateOverlapDefault(t *testing.T) {
	t.Parallel()
	if defaultRotateOverlap != 24*time.Hour {
		t.Fatalf("got %v", defaultRotateOverlap)
	}
}
