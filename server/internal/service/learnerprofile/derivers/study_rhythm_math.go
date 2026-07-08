package derivers

import (
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	studyRhythmWindowDays      = 90
	studyRhythmMinActiveDays   = 5
	studyRhythmSessionGap      = 30 * time.Minute
	studyRhythmHeartbeatSec    = 30
	studyRhythmDeriverVersion   = 1
	studyRhythmSourceTable      = "analytics.engagement_events"
	studyRhythmAuditSourceTable = "user.user_audit"
)

// PeakWindow is a dominant study time window in the learner's local timezone.
type PeakWindow struct {
	Dow        string  `json:"dow"`
	HourBucket string  `json:"hourBucket"`
	Share      float64 `json:"share"`
}

// RhythmSummary is the facet-level aggregate returned in summary JSON.
type RhythmSummary struct {
	PeakWindows           []PeakWindow `json:"peakWindows"`
	ConsistencyScore      float64      `json:"consistencyScore"`
	CurrentStreakDays     int          `json:"currentStreakDays"`
	LongestStreakDays     int          `json:"longestStreakDays"`
	MedianSessionMin      int          `json:"medianSessionMin"`
	SessionsPerActiveWeek float64      `json:"sessionsPerActiveWeek"`
	ActiveDaysPerWeek     float64      `json:"activeDaysPerWeek"`
}

type rhythmEvent struct {
	At       time.Time
	CourseID *string
}

type rhythmComputeInput struct {
	Events     []rhythmEvent
	ActiveDays []time.Time
	WindowStart time.Time
	WindowEnd   time.Time
	Now         time.Time
	Loc         *time.Location
}

func computeStudyRhythm(in rhythmComputeInput) (RhythmSummary, int, int) {
	loc := in.Loc
	if loc == nil {
		loc = time.UTC
	}

	eventTimes := make([]time.Time, 0, len(in.Events))
	for _, ev := range in.Events {
		eventTimes = append(eventTimes, ev.At)
	}
	sort.Slice(eventTimes, func(i, j int) bool { return eventTimes[i].Before(eventTimes[j]) })

	activeDays := normalizeDays(in.ActiveDays, loc)
	today := dateInLoc(in.Now, loc)

	currentStreak := currentStreakDays(activeDays, today)
	longestStreak := longestStreakDays(activeDays)
	peakWindows := peakStudyWindows(in.Events, loc)
	consistency, activeDaysPerWeek := consistencyMetrics(activeDays, in.WindowStart, in.WindowEnd, loc)
	medianMin, sessionCount, sessionsPerWeek := sessionMetrics(eventTimes, activeDays, in.WindowStart, in.WindowEnd, loc)

	return RhythmSummary{
		PeakWindows:           peakWindows,
		ConsistencyScore:      consistency,
		CurrentStreakDays:     currentStreak,
		LongestStreakDays:     longestStreak,
		MedianSessionMin:      medianMin,
		SessionsPerActiveWeek: sessionsPerWeek,
		ActiveDaysPerWeek:     activeDaysPerWeek,
	}, len(in.Events), sessionCount
}

func normalizeDays(days []time.Time, loc *time.Location) []time.Time {
	if len(days) == 0 {
		return nil
	}
	seen := make(map[string]time.Time, len(days))
	for _, d := range days {
		local := dateInLoc(d, loc)
		key := local.Format("2006-01-02")
		seen[key] = local
	}
	out := make([]time.Time, 0, len(seen))
	for _, d := range seen {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

func dateInLoc(t time.Time, loc *time.Location) time.Time {
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func dowCategory(weekday time.Weekday) string {
	if weekday == time.Saturday || weekday == time.Sunday {
		return "weekend"
	}
	return "weekday"
}

func hourBucket(hour int) string {
	start := (hour / 3) * 3
	end := start + 3
	if end > 24 {
		end = 24
	}
	return fmt.Sprintf("%d-%d", start, end)
}

func peakStudyWindows(events []rhythmEvent, loc *time.Location) []PeakWindow {
	if len(events) == 0 {
		return nil
	}
	type key struct {
		dow string
		hour string
	}
	counts := make(map[key]int)
	for _, ev := range events {
		local := ev.At.In(loc)
		k := key{dow: dowCategory(local.Weekday()), hour: hourBucket(local.Hour())}
		counts[k]++
	}
	best := key{}
	bestN := 0
	for k, n := range counts {
		if n > bestN {
			best, bestN = k, n
		}
	}
	if bestN == 0 {
		return nil
	}
	share := round2(float64(bestN) / float64(len(events)))
	return []PeakWindow{{
		Dow:        best.dow,
		HourBucket: best.hour,
		Share:      share,
	}}
}

func currentStreakDays(activeDays []time.Time, endDay time.Time) int {
	if len(activeDays) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(activeDays))
	for _, d := range activeDays {
		set[d.Format("2006-01-02")] = struct{}{}
	}
	streak := 0
	d := endDay
	for {
		if _, ok := set[d.Format("2006-01-02")]; !ok {
			break
		}
		streak++
		d = d.AddDate(0, 0, -1)
	}
	return streak
}

func longestStreakDays(activeDays []time.Time) int {
	if len(activeDays) == 0 {
		return 0
	}
	days := append([]time.Time(nil), activeDays...)
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	longest := 1
	run := 1
	for i := 1; i < len(days); i++ {
		gap := days[i].Sub(days[i-1])
		if gap == 24*time.Hour {
			run++
			if run > longest {
				longest = run
			}
			continue
		}
		run = 1
	}
	return longest
}

func consistencyMetrics(activeDays []time.Time, windowStart, windowEnd time.Time, loc *time.Location) (score float64, activeDaysPerWeek float64) {
	if len(activeDays) == 0 {
		return 0, 0
	}
	weeks := weeksInWindow(windowStart, windowEnd, loc)
	if weeks == 0 {
		weeks = 1
	}
	activeDaysPerWeek = round2(float64(len(activeDays)) / float64(weeks))

	weekly := weeklyActiveDayCounts(activeDays, windowStart, windowEnd, loc)
	if len(weekly) == 0 {
		return 0, activeDaysPerWeek
	}
	mean := 0.0
	for _, n := range weekly {
		mean += float64(n)
	}
	mean /= float64(len(weekly))
	if mean == 0 {
		return 0, activeDaysPerWeek
	}
	variance := 0.0
	for _, n := range weekly {
		diff := float64(n) - mean
		variance += diff * diff
	}
	variance /= float64(len(weekly))
	cv := math.Sqrt(variance) / mean
	score = round2(math.Max(0, 1-math.Min(1, cv)))
	return score, activeDaysPerWeek
}

func weeksInWindow(windowStart, windowEnd time.Time, loc *time.Location) int {
	start := dateInLoc(windowStart, loc)
	end := dateInLoc(windowEnd, loc)
	if !end.After(start) {
		return 1
	}
	days := int(end.Sub(start).Hours()/24) + 1
	weeks := days / 7
	if weeks < 1 {
		return 1
	}
	return weeks
}

func weekStartKey(day time.Time, loc *time.Location) string {
	local := day.In(loc)
	wd := int(local.Weekday())
	if wd == 0 {
		wd = 7
	}
	monday := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	monday = monday.AddDate(0, 0, -(wd - 1))
	return monday.Format("2006-01-02")
}

func weeklyActiveDayCounts(activeDays []time.Time, windowStart, windowEnd time.Time, loc *time.Location) []int {
	start := dateInLoc(windowStart, loc)
	end := dateInLoc(windowEnd, loc)
	byWeek := make(map[string]int)
	for _, d := range activeDays {
		if d.Before(start) || d.After(end) {
			continue
		}
		byWeek[weekStartKey(d, loc)]++
	}
	if len(byWeek) == 0 {
		return nil
	}
	out := make([]int, 0, len(byWeek))
	for _, n := range byWeek {
		out = append(out, n)
	}
	return out
}

func segmentSessions(events []time.Time, gap time.Duration) [][]time.Time {
	if len(events) == 0 {
		return nil
	}
	sessions := [][]time.Time{{events[0]}}
	for i := 1; i < len(events); i++ {
		if events[i].Sub(events[i-1]) > gap {
			sessions = append(sessions, []time.Time{events[i]})
		} else {
			sessions[len(sessions)-1] = append(sessions[len(sessions)-1], events[i])
		}
	}
	return sessions
}

func sessionLengthMinutes(heartbeatCount int) int {
	if heartbeatCount <= 0 {
		return 0
	}
	return int(math.Round(float64(heartbeatCount*studyRhythmHeartbeatSec) / 60.0))
}

func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return int(math.Round(float64(sorted[mid-1]+sorted[mid]) / 2.0))
}

func sessionMetrics(events []time.Time, activeDays []time.Time, windowStart, windowEnd time.Time, loc *time.Location) (medianMin int, sessionCount int, perActiveWeek float64) {
	sessions := segmentSessions(events, studyRhythmSessionGap)
	if len(sessions) == 0 {
		return 0, 0, 0
	}
	lengths := make([]int, len(sessions))
	for i, s := range sessions {
		lengths[i] = sessionLengthMinutes(len(s))
	}
	medianMin = medianInt(lengths)
	sessionCount = len(sessions)

	activeWeeks := len(weeklyActiveDayCounts(activeDays, windowStart, windowEnd, loc))
	if activeWeeks == 0 {
		activeWeeks = 1
	}
	perActiveWeek = round2(float64(sessionCount) / float64(activeWeeks))
	return medianMin, sessionCount, perActiveWeek
}

func rhythmConfidence(activeDayCount, eventCount int) float64 {
	if activeDayCount < studyRhythmMinActiveDays {
		return 0
	}
	c := math.Min(1, float64(activeDayCount)/20.0) * math.Min(1, float64(eventCount)/100.0)
	return round2(math.Max(0.25, c))
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func courseEventCounts(events []rhythmEvent) map[string]int {
	out := make(map[string]int)
	for _, ev := range events {
		key := ""
		if ev.CourseID != nil {
			key = *ev.CourseID
		}
		out[key]++
	}
	return out
}