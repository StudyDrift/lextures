package webhooksvc

import (
	"testing"
	"time"
)

func TestRetryDelaySchedule(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	expected := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		30 * time.Minute,
		2 * time.Hour,
		8 * time.Hour,
		24 * time.Hour,
	}
	for i, want := range expected {
		attempts := i + 1
		idx := attempts - 1
		if idx >= len(retryDelays) {
			idx = len(retryDelays) - 1
		}
		got := now.Add(retryDelays[idx])
		if !got.Equal(now.Add(want)) {
			t.Fatalf("attempt %d: want +%v got +%v", attempts, want, got.Sub(now))
		}
	}
}

func TestMaxAttemptsIsSix(t *testing.T) {
	if maxAttempts != 6 {
		t.Fatalf("maxAttempts=%d", maxAttempts)
	}
}
