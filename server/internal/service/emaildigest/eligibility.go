package emaildigest

import (
	"time"

	"github.com/lextures/lextures/server/internal/service/studyreminders"
)

const digestLookback = 25 * time.Hour

var digestSendTime = time.Date(0, 1, 1, 7, 0, 0, 0, time.UTC)

// ShouldSendDigest reports whether now is within the daily digest window (07:00 local).
func ShouldSendDigest(now time.Time, timezone *string) bool {
	localNow := studyreminders.UserLocalNow(now, timezone)
	localDate := studyreminders.UserLocalDate(now, timezone)
	target := studyreminders.ReminderClock(localDate, digestSendTime, timezone)
	return studyreminders.InReminderWindow(localNow, target)
}

// DigestSince returns the earliest created_at for digest items included in this send.
func DigestSince(now time.Time, timezone *string) time.Time {
	_ = timezone
	return now.Add(-digestLookback)
}

// LocalDayStartUTC is midnight at the start of the user's local calendar day.
func LocalDayStartUTC(now time.Time, timezone *string) time.Time {
	local := studyreminders.UserLocalNow(now, timezone)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
}