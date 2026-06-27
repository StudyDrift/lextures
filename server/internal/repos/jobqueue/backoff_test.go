package jobqueue

import (
	"testing"
	"time"
)

func TestBackoffDelay_Schedule(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 1 * time.Minute},  // clamped to 1
		{1, 1 * time.Minute},
		{2, 5 * time.Minute},
		{3, 30 * time.Minute},
		{4, 2 * time.Hour},
		{5, 8 * time.Hour},
		{6, 8 * time.Hour}, // beyond schedule reuses final delay
		{99, 8 * time.Hour},
	}
	for _, c := range cases {
		if got := BackoffDelay(c.attempt); got != c.want {
			t.Errorf("BackoffDelay(%d) = %s, want %s", c.attempt, got, c.want)
		}
	}
}

func TestNextRetryAt(t *testing.T) {
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	got := NextRetryAt(now, 2)
	want := now.Add(5 * time.Minute)
	if !got.Equal(want) {
		t.Fatalf("NextRetryAt = %s, want %s", got, want)
	}
}
