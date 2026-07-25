package outcomesreport

import "testing"

func TestMergeAdaptiveScore_Both(t *testing.T) {
	existing := float32(60)
	got := mergeAdaptiveScore(&existing, 80)
	if got == nil || *got != 70 {
		t.Fatalf("expected 70, got %v", got)
	}
}

func TestMergeAdaptiveScore_AdaptiveOnly(t *testing.T) {
	got := mergeAdaptiveScore(nil, 85)
	if got == nil || *got != 85 {
		t.Fatalf("expected 85, got %v", got)
	}
}
