package homeschoolrecords

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lextures/lextures/server/internal/repos/homeschoolrecords"
)

// SchoolYearWindow is an inclusive calendar range for a labeled school year.
type SchoolYearWindow struct {
	Label string // e.g. "2025-2026"
	From  time.Time
	To    time.Time // last inclusive day
}

// ParseSchoolYearLabel parses "YYYY-YYYY" (end year = start+1) or a single "YYYY".
func ParseSchoolYearLabel(label string) (startYear int, ok bool) {
	label = strings.TrimSpace(label)
	if label == "" {
		return 0, false
	}
	if parts := strings.Split(label, "-"); len(parts) == 2 {
		a, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		b, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil || b != a+1 || a < 1990 || a > 2100 {
			return 0, false
		}
		return a, true
	}
	y, err := strconv.Atoi(label)
	if err != nil || y < 1990 || y > 2100 {
		return 0, false
	}
	return y, true
}

// WindowForYear returns [start, day before next start) as inclusive dates using settings.
func WindowForYear(settings homeschoolrecords.Settings, startYear int) SchoolYearWindow {
	month := settings.SchoolYearStartMonth
	day := settings.SchoolYearStartDay
	if month < 1 || month > 12 {
		month = homeschoolrecords.DefaultSchoolYearStartMonth
	}
	if day < 1 || day > 31 {
		day = homeschoolrecords.DefaultSchoolYearStartDay
	}
	from := clampDate(startYear, time.Month(month), day)
	toExclusive := clampDate(startYear+1, time.Month(month), day)
	to := toExclusive.AddDate(0, 0, -1)
	return SchoolYearWindow{
		Label: fmt.Sprintf("%d-%d", startYear, startYear+1),
		From:  from,
		To:    to,
	}
}

// CurrentSchoolYear returns the window containing now.
func CurrentSchoolYear(settings homeschoolrecords.Settings, now time.Time) SchoolYearWindow {
	now = now.UTC()
	y := now.Year()
	candidate := WindowForYear(settings, y)
	if !now.Before(candidate.From) && !now.After(candidate.To.Add(24*time.Hour-time.Nanosecond)) {
		return candidate
	}
	// Before this year's start → previous year window.
	if now.Before(candidate.From) {
		return WindowForYear(settings, y-1)
	}
	return WindowForYear(settings, y)
}

func clampDate(year int, month time.Month, day int) time.Time {
	// Roll invalid days (e.g. Feb 31) to the last day of the month.
	t := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	last := t.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	if day < 1 {
		day = 1
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

