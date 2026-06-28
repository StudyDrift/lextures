package scheduler

import (
	"testing"
	"time"
)

func TestParseInvalid(t *testing.T) {
	cases := []string{
		"",
		"* * * *",     // 4 fields
		"* * * * * *", // 6 fields
		"60 * * * *",  // minute out of range
		"* 24 * * *",  // hour out of range
		"* * 0 * *",   // dom below range
		"* * * 13 *",  // month out of range
		"* * * * 7",   // dow out of range
		"*/0 * * * *", // zero step
		"5-1 * * * *", // inverted range
		"a * * * *",   // non-numeric
	}
	for _, c := range cases {
		if _, err := Parse(c); err == nil {
			t.Errorf("Parse(%q) expected error, got nil", c)
		}
	}
}

func TestNextDaily(t *testing.T) {
	s := MustParse("5 0 * * *") // 00:05 UTC daily
	from := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	got := s.Next(from)
	want := time.Date(2026, 6, 28, 0, 5, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Next = %v, want %v", got, want)
	}
}

func TestNextHourly(t *testing.T) {
	s := MustParse("0 * * * *")
	from := time.Date(2026, 6, 27, 12, 30, 0, 0, time.UTC)
	got := s.Next(from)
	want := time.Date(2026, 6, 27, 13, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Next = %v, want %v", got, want)
	}
}

func TestNextStrictlyAfter(t *testing.T) {
	// Exactly on a matching minute must return the *next* occurrence, not now.
	s := MustParse("0 * * * *")
	from := time.Date(2026, 6, 27, 13, 0, 0, 0, time.UTC)
	got := s.Next(from)
	want := time.Date(2026, 6, 27, 14, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Next = %v, want %v", got, want)
	}
}

func TestNextStepAndRange(t *testing.T) {
	s := MustParse("*/15 9-17 * * *") // every 15 min, 9am-5pm
	from := time.Date(2026, 6, 27, 9, 7, 0, 0, time.UTC)
	got := s.Next(from)
	want := time.Date(2026, 6, 27, 9, 15, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Next = %v, want %v", got, want)
	}
	// After the last slot of the day, it wraps to the next day's first slot.
	got = s.Next(time.Date(2026, 6, 27, 17, 50, 0, 0, time.UTC))
	want = time.Date(2026, 6, 28, 9, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Next wrap = %v, want %v", got, want)
	}
}

func TestNextDayOfWeek(t *testing.T) {
	s := MustParse("0 0 * * 1") // Mondays at midnight
	// 2026-06-27 is a Saturday; next Monday is 2026-06-29.
	got := s.Next(time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC))
	want := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("Next = %v, want %v", got, want)
	}
}

func TestIsDue(t *testing.T) {
	s := MustParse("0 * * * *") // hourly
	now := time.Date(2026, 6, 27, 13, 30, 0, 0, time.UTC)

	// Last run at 13:00; the 13:00 trigger already fired, next is 14:00 — not due.
	if s.IsDue(time.Date(2026, 6, 27, 13, 0, 0, 0, time.UTC), now) {
		t.Error("should not be due: last run this hour")
	}
	// Last run at 11:00 (app was down): 12:00 trigger missed, so due now (AC-5).
	if !s.IsDue(time.Date(2026, 6, 27, 11, 0, 0, 0, time.UTC), now) {
		t.Error("should be due: missed an interval")
	}
}

func TestBuiltinJobsCompile(t *testing.T) {
	jobs := BuiltinJobs()
	if len(jobs) != 5 {
		t.Fatalf("expected 5 builtin jobs, got %d", len(jobs))
	}
	seen := map[string]bool{}
	for _, j := range jobs {
		if seen[j.Name] {
			t.Errorf("duplicate job name %q", j.Name)
		}
		seen[j.Name] = true
		if j.JobType == "" {
			t.Errorf("job %q missing job type", j.Name)
		}
		// Next from a fixed time must resolve (schedule is valid & reachable).
		if j.Schedule().Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).IsZero() {
			t.Errorf("job %q schedule never fires", j.Name)
		}
	}
}
