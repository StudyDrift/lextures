package accommodations

import "testing"

func TestAppliedTimeLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		base       int32
		multiplier float64
		want       int32
	}{
		{0, 1.5, 0},
		{3600, 1, 3600},
		{3600, 1.5, 5400},
		{3600, 2, 7200},
		{100, 1.01, 101},
		{100, 0.5, 100},
	}
	for _, tc := range tests {
		got := AppliedTimeLimit(tc.base, tc.multiplier)
		if got != tc.want {
			t.Fatalf("AppliedTimeLimit(%d, %v) = %d, want %d", tc.base, tc.multiplier, got, tc.want)
		}
	}
}
