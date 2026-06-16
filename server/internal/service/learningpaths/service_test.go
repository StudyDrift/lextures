package learningpaths

import "testing"

func TestCalcProgressPercent(t *testing.T) {
	tests := []struct {
		completed, total, want int
	}{
		{0, 4, 0},
		{1, 4, 25},
		{3, 4, 75},
		{4, 4, 100},
		{0, 0, 0},
	}
	for _, tc := range tests {
		got := CalcProgressPercent(tc.completed, tc.total)
		if got != tc.want {
			t.Errorf("CalcProgressPercent(%d,%d) = %d, want %d", tc.completed, tc.total, got, tc.want)
		}
	}
}
