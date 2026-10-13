package parentprogress

import (
	"testing"
	"time"
)

func TestShouldSendRespectsTimezoneWeekdayAndHour(t *testing.T) {
	tz := "America/Denver"
	// 2026-10-11 is a Sunday. 15:00 UTC is 09:00 MDT.
	sundayMorning := time.Date(2026, 10, 11, 15, 0, 0, 0, time.UTC)
	local := LocalNow(sundayMorning, &tz)
	if !ShouldSend(local, time.Sunday, 8) {
		t.Fatalf("local %s should send on Sunday at or after 08:00", local)
	}
	if ShouldSend(local, time.Sunday, 10) {
		t.Fatal("should wait until the configured hour")
	}
	// 13:00 UTC Sunday is 07:00 MDT.
	tooEarly := LocalNow(time.Date(2026, 10, 11, 13, 0, 0, 0, time.UTC), &tz)
	if ShouldSend(tooEarly, time.Sunday, 8) {
		t.Fatal("07:00 local is before the send hour")
	}
	monday := LocalNow(time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC), &tz)
	if ShouldSend(monday, time.Sunday, 8) {
		t.Fatal("Monday is not the configured weekday")
	}
}

func TestNormalizeEmptyMode(t *testing.T) {
	if NormalizeEmptyMode("note") != EmptyNote || NormalizeEmptyMode(" NOTE ") != EmptyNote {
		t.Fatal("note mode")
	}
	if NormalizeEmptyMode("") != EmptySkip || NormalizeEmptyMode("skip") != EmptySkip {
		t.Fatal("skip mode")
	}
}

func TestWeekWindow(t *testing.T) {
	local := time.Date(2026, 10, 11, 9, 30, 0, 0, time.UTC)
	start, now, next := WeekWindow(local)
	if !start.Equal(time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("start %s", start)
	}
	if !now.Equal(local) || !next.Equal(local.AddDate(0, 0, 7)) {
		t.Fatalf("now %s next %s", now, next)
	}
}
