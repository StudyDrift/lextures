package attendancesessions

import "testing"

func TestPointsForStatus(t *testing.T) {
	tests := []struct {
		status   string
		points   int
		tardy    float64
		want     float64
	}{
		{"present", 10, 0.5, 10},
		{"excused", 10, 0.5, 10},
		{"tardy", 10, 0.5, 5},
		{"absent", 10, 0.5, 0},
		{"not_recorded", 10, 0.5, 0},
		{"present", 0, 0.5, 0},
	}
	for _, tc := range tests {
		got := PointsForStatus(tc.status, tc.points, tc.tardy)
		if got != tc.want {
			t.Errorf("PointsForStatus(%q, %d, %v) = %v, want %v", tc.status, tc.points, tc.tardy, got, tc.want)
		}
	}
}
