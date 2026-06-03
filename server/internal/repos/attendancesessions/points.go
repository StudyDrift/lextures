package attendancesessions

import "math"

// PointsForStatus returns gradebook points for a session status.
func PointsForStatus(status string, pointsPossible int, tardyRatio float64) float64 {
	if pointsPossible <= 0 {
		return 0
	}
	switch status {
	case "present", "excused":
		return float64(pointsPossible)
	case "tardy":
		return math.Round(float64(pointsPossible)*tardyRatio*100) / 100
	default:
		return 0
	}
}
