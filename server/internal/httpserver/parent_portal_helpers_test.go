package httpserver

import (
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/repos/attendance"
)

func TestParentAttendanceSummary(t *testing.T) {
	records := []attendance.Record{
		{Date: mustDate("2026-07-01"), Code: "P", CodeLabel: "Present", Category: "present"},
		{Date: mustDate("2026-06-30"), Code: "A", CodeLabel: "Absent", Category: "absent"},
		{Date: mustDate("2026-06-29"), Code: "T", CodeLabel: "Tardy", Category: "tardy"},
	}
	summary := parentAttendanceSummary(records, 2)
	if summary.Present != 1 || summary.Absent != 1 || summary.Tardy != 1 {
		t.Fatalf("unexpected counts: %+v", summary)
	}
	if len(summary.RecentDays) != 2 {
		t.Fatalf("recent days want 2 got %d", len(summary.RecentDays))
	}
	if summary.RecentDays[0].Date != "2026-07-01" {
		t.Fatalf("recent order: %+v", summary.RecentDays)
	}
}

func TestParentScorePercentage(t *testing.T) {
	pp := 100
	pct := parentScorePercentage("85", &pp)
	if pct == nil || *pct != 85 {
		t.Fatalf("expected 85 got %#v", pct)
	}
}

func TestParentGradeStatus(t *testing.T) {
	now := time.Now()
	if parentGradeStatus(true, &now) != "excused" {
		t.Fatal("excused")
	}
	if parentGradeStatus(false, &now) != "posted" {
		t.Fatal("posted")
	}
	if parentGradeStatus(false, nil) != "graded" {
		t.Fatal("graded")
	}
}

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
