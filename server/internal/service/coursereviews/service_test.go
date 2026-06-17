package coursereviews

import (
	"testing"
	"time"
)

func TestWithinEditWindow(t *testing.T) {
	created := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if !withinEditWindow(created, created.Add(29*24*time.Hour)) {
		t.Fatal("expected editable within 29 days")
	}
	if withinEditWindow(created, created.Add(31*24*time.Hour)) {
		t.Fatal("expected not editable after 31 days")
	}
}

func TestMinCompletionPercentConstant(t *testing.T) {
	if MinCompletionPercent != 10 {
		t.Fatalf("want 10 got %d", MinCompletionPercent)
	}
}

func TestProgressEligibilityThreshold(t *testing.T) {
	cases := []struct {
		completed, total int
		eligible         bool
	}{
		{0, 10, false},
		{1, 10, true},
		{2, 20, true},
		{1, 20, false},
	}
	for _, tc := range cases {
		pct := 0
		if tc.total > 0 && tc.completed > 0 {
			pct = int(float64(tc.completed) / float64(tc.total) * 100)
		}
		got := pct >= MinCompletionPercent
		if got != tc.eligible {
			t.Fatalf("%d/%d => %d%% eligible=%v want %v", tc.completed, tc.total, pct, got, tc.eligible)
		}
	}
}
