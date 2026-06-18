package studyreminders

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/service/gamification"
)

const reminderWindow = 5 * time.Minute

// UserLocalNow returns now in the user's timezone.
func UserLocalNow(now time.Time, timezone *string) time.Time {
	loc := time.UTC
	if timezone != nil && *timezone != "" {
		if l, err := time.LoadLocation(*timezone); err == nil {
			loc = l
		}
	}
	return now.In(loc)
}

// UserLocalDate is the calendar date for now in the user's timezone (UTC midnight encoding).
func UserLocalDate(now time.Time, timezone *string) time.Time {
	local := UserLocalNow(now, timezone)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

// ReminderClock builds today's reminder instant in the user's timezone.
func ReminderClock(localDate time.Time, reminderTime time.Time, timezone *string) time.Time {
	loc := time.UTC
	if timezone != nil && *timezone != "" {
		if l, err := time.LoadLocation(*timezone); err == nil {
			loc = l
		}
	}
	y, m, d := localDate.Year(), localDate.Month(), localDate.Day()
	return time.Date(y, m, d, reminderTime.Hour(), reminderTime.Minute(), 0, 0, loc)
}

// InReminderWindow is true when localNow is within [target, target+window).
func InReminderWindow(localNow, target time.Time) bool {
	return !localNow.Before(target) && localNow.Before(target.Add(reminderWindow))
}

// ShouldSendDaily returns whether the daily reminder window is active.
func ShouldSendDaily(localNow time.Time, localDate time.Time, reminderTime time.Time, timezone *string, studiedToday bool) bool {
	if studiedToday {
		return false
	}
	target := ReminderClock(localDate, reminderTime, timezone)
	return InReminderWindow(localNow, target)
}

// ShouldSendStreakAtRisk returns whether the streak-at-risk window is active (2h before reminder).
func ShouldSendStreakAtRisk(localNow time.Time, localDate time.Time, reminderTime time.Time, timezone *string, studiedToday bool, currentStreak int) bool {
	if studiedToday || currentStreak <= 0 {
		return false
	}
	target := ReminderClock(localDate, reminderTime, timezone).Add(-2 * time.Hour)
	return InReminderWindow(localNow, target)
}

// ShouldSendWeeklySummary returns true on Sunday at the reminder time when weekly summary is enabled.
func ShouldSendWeeklySummary(localNow time.Time, localDate time.Time, reminderTime time.Time, timezone *string, weeklySummary bool) bool {
	if !weeklySummary {
		return false
	}
	if localNow.Weekday() != time.Sunday {
		return false
	}
	target := ReminderClock(localDate, reminderTime, timezone)
	return InReminderWindow(localNow, target)
}

// StreakAtRiskBanner is true when local hour > 18 and the learner has not studied today with an active streak.
func StreakAtRiskBanner(localNow time.Time, today time.Time, currentStreak int, lastActivity *time.Time, timezone *string) bool {
	if currentStreak <= 0 {
		return false
	}
	if localNow.Hour() < 18 {
		return false
	}
	atRisk, _ := gamification.StreakAtRisk(currentStreak, lastActivity, today, localNow.UTC(), timezone)
	return atRisk
}

// IdempotencyKey builds a stable deduplication key.
func IdempotencyKey(userID uuid.UUID, localDate time.Time, reminderType, channel string) string {
	return fmt.Sprintf("%s|%s|%s|%s", userID.String(), localDate.Format("2006-01-02"), reminderType, channel)
}
