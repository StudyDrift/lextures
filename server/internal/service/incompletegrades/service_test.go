package incompletegrades

import (
	"testing"
	"time"
)

func TestDateOnly(t *testing.T) {
	in := time.Date(2026, 6, 8, 15, 30, 0, 0, time.UTC)
	got := dateOnly(in)
	want := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("dateOnly: got %v want %v", got, want)
	}
}
