package parentprogress

import (
	"strings"
	"time"
)

const (
	EmptySkip = "skip"
	EmptyNote = "note"
)

// LocalNow returns now in the user's IANA timezone, or UTC when unset or invalid.
func LocalNow(now time.Time, timezone *string) time.Time {
	loc := time.UTC
	if timezone != nil {
		if name := strings.TrimSpace(*timezone); name != "" {
			if loaded, err := time.LoadLocation(name); err == nil {
				loc = loaded
			}
		}
	}
	return now.In(loc)
}

// NormalizeWeekday clamps a config value to Sunday–Saturday. Out of range becomes Sunday.
func NormalizeWeekday(n int) time.Weekday {
	if n < 0 || n > 6 {
		return time.Sunday
	}
	return time.Weekday(n)
}

// NormalizeHour clamps a config hour to 0–23. Out of range becomes 8.
func NormalizeHour(n int) int {
	if n < 0 || n > 23 {
		return 8
	}
	return n
}

// NormalizeEmptyMode accepts "note" or "skip".
func NormalizeEmptyMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), EmptyNote) {
		return EmptyNote
	}
	return EmptySkip
}

// ShouldSend reports whether local time is on the configured weekday at or after the hour.
func ShouldSend(local time.Time, weekday time.Weekday, hour int) bool {
	return local.Weekday() == weekday && local.Hour() >= hour
}

// WeekWindow is the previous seven local midnights through local now, and the next seven days.
func WeekWindow(local time.Time) (pastStart, now, nextEnd time.Time) {
	loc := local.Location()
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return midnight.AddDate(0, 0, -7), local, local.AddDate(0, 0, 7)
}
