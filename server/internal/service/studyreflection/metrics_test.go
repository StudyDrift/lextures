package studyreflection

import (
	"testing"
	"time"
)

func TestWeekBoundsMonday(t *testing.T) {
	// 2026-05-21 is Thursday
	thu := time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC)
	start, end := WeekBounds(thu)
	if start.Weekday() != time.Monday || start.Day() != 18 {
		t.Fatalf("start=%v want Mon May 18", start)
	}
	if end.Day() != 24 || end.Hour() != 23 {
		t.Fatalf("end=%v want Sun May 24 end of day", end)
	}
}

func TestLoginStreak(t *testing.T) {
	end := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	days := map[string]struct{}{
		"2026-05-19": {},
		"2026-05-20": {},
		"2026-05-21": {},
	}
	if got := LoginStreak(days, end); got != 3 {
		t.Fatalf("streak=%d want 3", got)
	}
	days["2026-05-18"] = struct{}{}
	if got := LoginStreak(days, end); got != 4 {
		t.Fatalf("streak=%d want 4", got)
	}
	if got := LoginStreak(nil, end); got != 0 {
		t.Fatalf("streak=%d want 0", got)
	}
}

func TestStudyEfficiency(t *testing.T) {
	start, end := 70.0, 78.0
	r, low, ok := StudyEfficiency(3600, &start, &end)
	if !ok || r <= 0 {
		t.Fatalf("ok=%v ratio=%v", ok, r)
	}
	flatEnd := 71.0
	_, low, ok = StudyEfficiency(8000, &start, &flatEnd)
	if !ok || !low {
		t.Fatalf("expected low efficiency flag when time is high but scores barely improve")
	}
	_, _, ok = StudyEfficiency(0, &start, &end)
	if ok {
		t.Fatal("expected not ok with zero time")
	}
}
