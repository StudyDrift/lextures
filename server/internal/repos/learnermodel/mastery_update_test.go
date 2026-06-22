package learnermodel

import (
	"math"
	"testing"
	"time"
)

func TestDecayAdjustedMastery_noDecayWithoutLastSeen(t *testing.T) {
	got := DecayAdjustedMastery(0.7, nil, 0.02)
	if math.Abs(got-0.7) > 1e-9 {
		t.Fatalf("got %v want 0.7", got)
	}
}

func TestDecayAdjustedMastery_appliesDecay(t *testing.T) {
	last := time.Now().UTC().Add(-10 * 24 * time.Hour)
	got := DecayAdjustedMastery(0.8, &last, 0.02)
	want := 0.8 * math.Exp(-0.02*10)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}