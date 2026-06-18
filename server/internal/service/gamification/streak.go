package gamification

import (
	"time"
)

// UserLocalDate returns the calendar date for t in the user's IANA timezone, or UTC when unset/invalid.
func UserLocalDate(t time.Time, timezone *string) time.Time {
	loc := time.UTC
	if timezone != nil && *timezone != "" {
		if l, err := time.LoadLocation(*timezone); err == nil {
			loc = l
		}
	}
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// StreakUpdateResult describes how a streak changed after activity or reconciliation.
type StreakUpdateResult struct {
	CurrentStreak int
	LongestStreak int
	StreakEnded   bool
	FreezeUsed    bool
}

// ComputeStreakAfterActivity updates streak counters when the learner completes an activity today.
func ComputeStreakAfterActivity(
	lastActivityDate *time.Time,
	currentStreak, longestStreak int,
	freezeCoverDate *time.Time,
	today time.Time,
) (newStreak, newLongest int, freezeConsumed bool) {
	if lastActivityDate == nil {
		return 1, max(longestStreak, 1), false
	}
	last := time.Date(lastActivityDate.Year(), lastActivityDate.Month(), lastActivityDate.Day(), 0, 0, 0, 0, time.UTC)
	if last.Equal(today) {
		return currentStreak, longestStreak, false
	}
	yesterday := today.AddDate(0, 0, -1)
	if last.Equal(yesterday) {
		newStreak = currentStreak + 1
		return newStreak, max(longestStreak, newStreak), false
	}
	if freezeCoverDate != nil {
		cover := time.Date(freezeCoverDate.Year(), freezeCoverDate.Month(), freezeCoverDate.Day(), 0, 0, 0, 0, time.UTC)
		if cover.Equal(yesterday) && last.Equal(yesterday.AddDate(0, 0, -1)) {
			newStreak = currentStreak + 1
			return newStreak, max(longestStreak, newStreak), true
		}
	}
	return 1, max(longestStreak, 1), false
}

// ReconcileStreakOnLogin checks whether the streak should reset when the user has not been active recently.
func ReconcileStreakOnLogin(
	lastActivityDate *time.Time,
	currentStreak int,
	freezeCoverDate *time.Time,
	today time.Time,
) (newStreak int, streakEnded bool, freezeConsumed bool) {
	if lastActivityDate == nil || currentStreak == 0 {
		return 0, false, false
	}
	last := time.Date(lastActivityDate.Year(), lastActivityDate.Month(), lastActivityDate.Day(), 0, 0, 0, 0, time.UTC)
	if !last.Before(today) {
		return currentStreak, false, false
	}
	yesterday := today.AddDate(0, 0, -1)
	if last.Equal(yesterday) {
		return currentStreak, false, false
	}
	if freezeCoverDate != nil {
		cover := time.Date(freezeCoverDate.Year(), freezeCoverDate.Month(), freezeCoverDate.Day(), 0, 0, 0, 0, time.UTC)
		if cover.Equal(yesterday) {
			return currentStreak, false, true
		}
	}
	return 0, currentStreak > 0, false
}

// StreakAtRisk reports whether the learner must complete an activity before endOfDay to keep their streak.
func StreakAtRisk(currentStreak int, lastActivityDate *time.Time, today time.Time, now time.Time, timezone *string) (atRisk bool, hoursLeft float64) {
	if currentStreak <= 0 || lastActivityDate == nil {
		return false, 0
	}
	last := time.Date(lastActivityDate.Year(), lastActivityDate.Month(), lastActivityDate.Day(), 0, 0, 0, 0, time.UTC)
	if last.Equal(today) {
		return false, 0
	}
	yesterday := today.AddDate(0, 0, -1)
	if !last.Equal(yesterday) {
		return false, 0
	}
	loc := time.UTC
	if timezone != nil && *timezone != "" {
		if l, err := time.LoadLocation(*timezone); err == nil {
			loc = l
		}
	}
	localNow := now.In(loc)
	endOfDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 23, 59, 59, 999999999, loc)
	hoursLeft = endOfDay.Sub(localNow).Hours()
	return hoursLeft > 0 && hoursLeft <= 24, hoursLeft
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
