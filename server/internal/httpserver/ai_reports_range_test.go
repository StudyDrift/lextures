package httpserver

import (
	"net/url"
	"testing"
	"time"
)

func TestParseAIReportsTimeRange_defaults24h(t *testing.T) {
	now, err := time.Parse(time.RFC3339, "2026-06-16T12:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	from, to, err := parseAIReportsTimeRange(url.Values{}, now)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !to.Equal(now) {
		t.Fatalf("to: got %v", to)
	}
	if got := to.Sub(from); got != 24*time.Hour {
		t.Fatalf("from default 24h before to: from=%v to=%v sub=%v", from, to, got)
	}
}

func TestParseAIReportsTimeRange_explicit(t *testing.T) {
	now := time.Unix(0, 0).UTC()
	v := url.Values{}
	v.Set("from", "2026-01-01T00:00:00Z")
	v.Set("to", "2026-01-20T00:00:00Z")
	from, to, err := parseAIReportsTimeRange(v, now)
	if err != nil {
		t.Fatal(err)
	}
	if from.Format(time.RFC3339) != "2026-01-01T00:00:00Z" || to.Format(time.RFC3339) != "2026-01-20T00:00:00Z" {
		t.Fatalf("got from=%v to=%v", from, to)
	}
}