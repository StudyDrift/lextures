package httpserver

import (
	"testing"
	"time"

	"github.com/google/uuid"
	stprog "github.com/lextures/lextures/server/internal/repos/studentprogress"
)

func TestParentHomeWeek_MondayStartInZone(t *testing.T) {
	loc, err := time.LoadLocation("America/Boise")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	// Sunday 2026-10-11 23:30 Boise is still the week that began Monday 2026-10-05.
	now := time.Date(2026, 10, 11, 23, 30, 0, 0, loc)
	start, end := parentHomeWeek(now, loc)
	if got := start.In(loc); got.Weekday() != time.Monday || got.Day() != 5 || got.Hour() != 0 {
		t.Fatalf("start = %v", got)
	}
	if end.Sub(start) != 7*24*time.Hour {
		t.Fatalf("week length = %v", end.Sub(start))
	}
	// Monday 00:10 starts a new week.
	start2, _ := parentHomeWeek(time.Date(2026, 10, 12, 0, 10, 0, 0, loc), loc)
	if start2.In(loc).Day() != 12 {
		t.Fatalf("start2 = %v", start2.In(loc))
	}
	if s, _ := parentHomeWeek(now, nil); s.Location() != time.UTC {
		t.Fatalf("nil loc should fall back to UTC")
	}
}

func TestClassifyParentHomeWork(t *testing.T) {
	weekStart := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	weekEnd := weekStart.AddDate(0, 0, 7)
	now := weekStart.Add(50 * time.Hour) // Wednesday
	at := func(h int) *time.Time { v := weekStart.Add(time.Duration(h) * time.Hour); return &v }
	ids := make([]uuid.UUID, 7)
	for i := range ids {
		ids[i] = uuid.New()
	}
	rows := []stprog.WorkItemRow{
		{ItemID: ids[0], DueAt: at(100)},                       // due later this week
		{ItemID: ids[1], DueAt: at(10)},                        // due earlier this week, overdue
		{ItemID: ids[2], DueAt: at(-200)},                      // overdue from last week
		{ItemID: ids[3], DueAt: at(300)},                       // next week: excluded
		{ItemID: ids[4], DueAt: at(20), CompletedAt: at(19)},   // completed this week
		{ItemID: ids[5], DueAt: at(-50), CompletedAt: at(-60)}, // completed last week: excluded
		{ItemID: ids[6]},                                       // no due date, incomplete: excluded
	}
	due, done, overdue := classifyParentHomeWork(rows, now, weekStart, weekEnd)
	if len(due) != 3 || due[0].ItemID != ids[0] || due[1].ItemID != ids[1] || due[2].ItemID != ids[2] {
		t.Fatalf("due = %+v", due)
	}
	if overdue[ids[0]] || !overdue[ids[1]] || !overdue[ids[2]] {
		t.Fatalf("overdue = %+v", overdue)
	}
	if len(done) != 1 || done[0].ItemID != ids[4] {
		t.Fatalf("done = %+v", done)
	}
}
