package homeschoolrecords

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/homeschoolrecords"
)

func TestWindowForYear_DefaultAugust(t *testing.T) {
	s := homeschoolrecords.DefaultSettings(uuid.New())
	w := WindowForYear(s, 2025)
	if w.Label != "2025-2026" {
		t.Fatalf("label = %s", w.Label)
	}
	if w.From.Format("2006-01-02") != "2025-08-01" {
		t.Fatalf("from = %s", w.From)
	}
	if w.To.Format("2006-01-02") != "2026-07-31" {
		t.Fatalf("to = %s", w.To)
	}
}

func TestCurrentSchoolYear(t *testing.T) {
	s := homeschoolrecords.DefaultSettings(uuid.New())
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	w := CurrentSchoolYear(s, now)
	if w.Label != "2025-2026" {
		t.Fatalf("got %s", w.Label)
	}
	now2 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	w2 := CurrentSchoolYear(s, now2)
	if w2.Label != "2026-2027" {
		t.Fatalf("got %s", w2.Label)
	}
}

func TestParseSchoolYearLabel(t *testing.T) {
	y, ok := ParseSchoolYearLabel("2025-2026")
	if !ok || y != 2025 {
		t.Fatalf("%v %v", y, ok)
	}
	if _, ok := ParseSchoolYearLabel("2025-2027"); ok {
		t.Fatal("expected reject")
	}
}

func TestBuildHoursLog_MergesAndTotals(t *testing.T) {
	from := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 8, 31, 0, 0, 0, 0, time.UTC)
	cid := uuid.New()
	auto := []homeschoolrecords.AutoHoursRow{{
		EntryDate: time.Date(2025, 8, 10, 0, 0, 0, 0, time.UTC),
		Subject:   "Math",
		CourseID:  &cid,
		Hours:     1.5,
	}}
	manual := []homeschoolrecords.HoursEntry{{
		ID:        uuid.New(),
		EntryDate: time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC),
		Subject:   "Math",
		Hours:     2,
		Source:    "offline",
		Notes:     "Museum",
	}}
	log := BuildHoursLog(from, to, "2025-2026", auto, manual)
	if len(log.Entries) != 2 {
		t.Fatalf("entries = %d", len(log.Entries))
	}
	if log.TotalHours != 3.5 {
		t.Fatalf("total = %v", log.TotalHours)
	}
	if len(log.TotalsBySubject) != 1 || log.TotalsBySubject[0].Hours != 3.5 {
		t.Fatalf("totals = %+v", log.TotalsBySubject)
	}
	if !log.Entries[0].Editable && log.Entries[1].Editable {
		// order is date desc: Aug 11 then Aug 10
	}
	if log.Entries[0].Date != "2025-08-11" || !log.Entries[0].Editable {
		t.Fatalf("first = %+v", log.Entries[0])
	}
}
