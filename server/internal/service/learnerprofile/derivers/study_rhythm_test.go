package derivers

import (
	"testing"
	"time"
)

func TestSegmentSessions_30MinGap(t *testing.T) {
	base := time.Date(2026, 3, 1, 18, 0, 0, 0, time.UTC)
	events := []time.Time{
		base,
		base.Add(5 * time.Minute),
		base.Add(10 * time.Minute),
		base.Add(45 * time.Minute),
		base.Add(50 * time.Minute),
	}
	sessions := segmentSessions(events, studyRhythmSessionGap)
	if len(sessions) != 2 {
		t.Fatalf("sessions=%d want 2", len(sessions))
	}
	if len(sessions[0]) != 3 || len(sessions[1]) != 2 {
		t.Fatalf("unexpected session sizes: %+v", sessions)
	}
}

func TestSessionLengthMinutes_Median35(t *testing.T) {
	// 70 heartbeats at 30s each ≈ 35 minutes.
	lengths := []int{sessionLengthMinutes(70)}
	if got := medianInt(lengths); got < 34 || got > 36 {
		t.Fatalf("median=%d want ~35", got)
	}
}

func TestLongestStreak_12Days(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	days := make([]time.Time, 12)
	for i := range days {
		days[i] = start.AddDate(0, 0, i)
	}
	if got := longestStreakDays(days); got < 12 {
		t.Fatalf("longest=%d want >= 12", got)
	}
}

func TestCurrentStreak_ResetsAfterGap(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	days := make([]time.Time, 12)
	for i := range days {
		days[i] = start.AddDate(0, 0, i)
	}
	endDay := time.Date(2026, 1, 20, 0, 0, 0, 0, loc)
	if got := currentStreakDays(days, endDay); got != 0 {
		t.Fatalf("current after gap=%d want 0", got)
	}
	endDay = time.Date(2026, 1, 12, 0, 0, 0, 0, loc)
	if got := currentStreakDays(days, endDay); got != 12 {
		t.Fatalf("current=%d want 12", got)
	}
}

func TestPeakStudyWindows_EveningWeekday(t *testing.T) {
	loc := time.UTC
	events := make([]rhythmEvent, 0, 20)
	for i := 0; i < 20; i++ {
		// Tuesday 7–10pm UTC.
		at := time.Date(2026, 3, 3, 19, 0, 0, 0, loc).Add(time.Duration(i) * time.Minute)
		events = append(events, rhythmEvent{At: at})
	}
	for i := 0; i < 5; i++ {
		at := time.Date(2026, 3, 7, 10, 0, 0, 0, loc).Add(time.Duration(i) * time.Minute)
		events = append(events, rhythmEvent{At: at})
	}
	peaks := peakStudyWindows(events, loc)
	if len(peaks) == 0 {
		t.Fatal("expected peak window")
	}
	if peaks[0].Dow != "weekday" {
		t.Fatalf("dow=%q want weekday", peaks[0].Dow)
	}
	if peaks[0].HourBucket != "18-21" {
		t.Fatalf("hourBucket=%q want 18-21", peaks[0].HourBucket)
	}
}

func TestPeakStudyWindows_TimezoneAwareDenver(t *testing.T) {
	loc, err := time.LoadLocation("America/Denver")
	if err != nil {
		t.Fatal(err)
	}
	// 2026-01-15 02:00 UTC is 2026-01-14 19:00 in Denver (MST).
	events := []rhythmEvent{{At: time.Date(2026, 1, 15, 2, 0, 0, 0, time.UTC)}}
	peaks := peakStudyWindows(events, loc)
	if len(peaks) == 0 {
		t.Fatal("expected peak window")
	}
	if peaks[0].HourBucket != "18-21" {
		t.Fatalf("hourBucket=%q want 18-21 for local 7pm", peaks[0].HourBucket)
	}
}

func TestComputeStudyRhythm_InsufficientActiveDays(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, loc)
	windowStart := now.AddDate(0, 0, -studyRhythmWindowDays)
	days := []time.Time{
		now.AddDate(0, 0, -1),
		now.AddDate(0, 0, -3),
	}
	if len(days) >= studyRhythmMinActiveDays {
		t.Fatal("test setup should be below threshold")
	}
	_, eventCount, _ := computeStudyRhythm(rhythmComputeInput{
		Events:      nil,
		ActiveDays:  days,
		WindowStart: windowStart,
		WindowEnd:   now,
		Now:         now,
		Loc:         loc,
	})
	if eventCount != 0 {
		t.Fatalf("eventCount=%d", eventCount)
	}
}

func TestComputeStudyRhythm_MedianSessionApprox35(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 4, 10, 22, 0, 0, 0, loc)
	windowStart := now.AddDate(0, 0, -30)
	activeDays := make([]time.Time, 6)
	for i := range activeDays {
		activeDays[i] = now.AddDate(0, 0, -(i+1))
	}
	base := now.AddDate(0, 0, -2)
	events := make([]rhythmEvent, 70)
	for i := range events {
		events[i] = rhythmEvent{At: base.Add(time.Duration(i) * 30 * time.Second)}
	}
	summary, _, sessionCount := computeStudyRhythm(rhythmComputeInput{
		Events:      events,
		ActiveDays:  activeDays,
		WindowStart: windowStart,
		WindowEnd:   now,
		Loc:         loc,
		Now:         now,
	})
	if sessionCount != 1 {
		t.Fatalf("sessionCount=%d want 1", sessionCount)
	}
	if summary.MedianSessionMin < 34 || summary.MedianSessionMin > 36 {
		t.Fatalf("medianSessionMin=%d want ~35", summary.MedianSessionMin)
	}
}

func TestRhythmConfidence_BelowMinActiveDays(t *testing.T) {
	if got := rhythmConfidence(4, 100); got != 0 {
		t.Fatalf("confidence=%v want 0", got)
	}
}