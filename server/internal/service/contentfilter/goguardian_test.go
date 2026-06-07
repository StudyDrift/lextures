package contentfilter

import (
	"testing"

	"github.com/google/uuid"
)

func TestStudentIDHash_Deterministic(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	h1 := StudentIDHash(id, "district-salt")
	h2 := StudentIDHash(id, "district-salt")
	if h1 != h2 {
		t.Fatalf("expected deterministic hash, got %q vs %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(h1))
	}
}

func TestStudentIDHash_NotReversible(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	h := StudentIDHash(id, "salt")
	if h == id.String() {
		t.Fatal("hash must not equal raw UUID")
	}
}

func TestActivityEventJSON(t *testing.T) {
	ev := ActivityEvent{
		URL:           "https://district.lextures.com/courses/math",
		Category:      "educational",
		Title:         "Math 101",
		StudentIDHash: "abc123",
	}
	if ev.Category != "educational" {
		t.Fatalf("expected educational category, got %q", ev.Category)
	}
}
