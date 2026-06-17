package httpserver

import (
	"fmt"
	"net/url"
	"time"
)

const (
	aiReportsDefaultRange = 24 * time.Hour
	aiReportsMaxRangeDays = int64(366)
)

// parseAIReportsTimeRange resolves [from, to) for Intelligence AI reports (default: last 24 hours).
func parseAIReportsTimeRange(q url.Values, now time.Time) (from, to time.Time, err error) {
	to = now
	if s := q.Get("to"); s != "" {
		t, perr := time.Parse(time.RFC3339, s)
		if perr != nil {
			return time.Time{}, time.Time{}, errAIReportsTimeRangeInvalid()
		}
		to = t.UTC()
	}
	from = to.Add(-aiReportsDefaultRange)
	if s := q.Get("from"); s != "" {
		f, perr := time.Parse(time.RFC3339, s)
		if perr != nil {
			return time.Time{}, time.Time{}, errAIReportsTimeRangeInvalid()
		}
		from = f.UTC()
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("`from` must be before `to`.")
	}
	sec := to.Unix() - from.Unix()
	days := sec / 86400
	if days > aiReportsMaxRangeDays {
		return time.Time{}, time.Time{}, fmt.Errorf("Date range cannot exceed %d days.", aiReportsMaxRangeDays)
	}
	return from, to, nil
}

func errAIReportsTimeRangeInvalid() error {
	return fmt.Errorf("Invalid `from` or `to`: use RFC 3339 (e.g. 2026-04-01T00:00:00Z).")
}