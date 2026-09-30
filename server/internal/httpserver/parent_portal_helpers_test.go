package httpserver

import (
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/repos/attendance"
)

func TestParentAttendanceSummary(t *testing.T) {
	// Use dates relative to now so the rolling 3-month term window stays valid.
	now := time.Now().UTC()
	d0 := now.AddDate(0, 0, -1)
	d1 := now.AddDate(0, 0, -2)
	d2 := now.AddDate(0, 0, -3)
	records := []attendance.Record{
		{Date: d0, Code: "P", CodeLabel: "Present", Category: "present"},
		{Date: d1, Code: "A", CodeLabel: "Absent", Category: "absent"},
		{Date: d2, Code: "T", CodeLabel: "Tardy", Category: "tardy"},
	}
	summary := parentAttendanceSummary(records, 2)
	if summary.Present != 1 || summary.Absent != 1 || summary.Tardy != 1 {
		t.Fatalf("unexpected counts: %+v", summary)
	}
	if len(summary.RecentDays) != 2 {
		t.Fatalf("recent days want 2 got %d", len(summary.RecentDays))
	}
	if summary.RecentDays[0].Date != d0.Format("2006-01-02") {
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

