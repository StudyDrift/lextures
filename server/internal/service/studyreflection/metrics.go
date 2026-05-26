package studyreflection

import "time"

// WeekBounds returns Monday 00:00 UTC through Sunday 23:59:59.999 for the week containing t.
func WeekBounds(t time.Time) (start, end time.Time) {
	utc := t.UTC()
	wd := int(utc.Weekday())
	if wd == 0 {
		wd = 7
	}
	start = time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	start = start.AddDate(0, 0, -(wd - 1))
	end = start.AddDate(0, 0, 7).Add(-time.Nanosecond)
	return start, end
}

// LoginStreak counts consecutive calendar days ending on endDay (inclusive) where activeDays is set.
func LoginStreak(activeDays map[string]struct{}, endDay time.Time) int {
	if len(activeDays) == 0 {
		return 0
	}
	streak := 0
	d := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 0, 0, 0, 0, time.UTC)
	for {
		key := d.Format("2006-01-02")
		if _, ok := activeDays[key]; !ok {
			break
		}
		streak++
		d = d.AddDate(0, 0, -1)
	}
	return streak
}

// StudyEfficiency returns quiz score improvement per hour on task; ok false when insufficient data.
func StudyEfficiency(timeOnTaskSeconds int, scoreStart, scoreEnd *float64) (ratio float64, lowEfficiency bool, ok bool) {
	if timeOnTaskSeconds <= 0 || scoreStart == nil || scoreEnd == nil {
		return 0, false, false
	}
	improvement := *scoreEnd - *scoreStart
	hours := float64(timeOnTaskSeconds) / 3600.0
	if hours <= 0 {
		return 0, false, false
	}
	ratio = improvement / hours
	ok = true
	// Flag when >2h studied but improvement under 2 points (on 0–100 scale).
	lowEfficiency = timeOnTaskSeconds >= 7200 && improvement < 2
	return ratio, lowEfficiency, ok
}
