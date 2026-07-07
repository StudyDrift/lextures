package cli

import (
	"fmt"
	"time"
)

// ResolveLocation returns the timezone location for flag value or local default.
func ResolveLocation(tz string) (*time.Location, error) {
	tz = trim(tz)
	if tz == "" {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	return loc, nil
}

// ParseRFC3339InTZ parses an RFC3339 time or a local datetime in the given timezone.
func ParseRFC3339InTZ(raw, tz string) (time.Time, error) {
	raw = trim(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	loc, err := ResolveLocation(tz)
	if err != nil {
		return time.Time{}, err
	}
	layouts := []string{"2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse time %q (use RFC3339 or YYYY-MM-DD HH:MM with --tz)", raw)
}

// FormatRFC3339 formats t in the given timezone for API payloads.
func FormatRFC3339(t time.Time, tz string) (string, error) {
	loc, err := ResolveLocation(tz)
	if err != nil {
		return "", err
	}
	return t.In(loc).Format(time.RFC3339), nil
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}