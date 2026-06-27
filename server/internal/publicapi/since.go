package publicapi

import (
	"net/url"
	"strings"
	"time"
)

// ParseSinceTime reads an optional RFC3339 `since` query parameter for polling triggers.
func ParseSinceTime(q url.Values) (*time.Time, error) {
	raw := strings.TrimSpace(q.Get("since"))
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	utc := t.UTC()
	return &utc, nil
}

// FilterBySince keeps items whose timestamp is strictly after since.
func FilterBySince[T any](items []T, since *time.Time, ts func(T) *time.Time) []T {
	if since == nil {
		return items
	}
	out := make([]T, 0, len(items))
	for _, item := range items {
		t := ts(item)
		if t != nil && t.After(*since) {
			out = append(out, item)
		}
	}
	return out
}
