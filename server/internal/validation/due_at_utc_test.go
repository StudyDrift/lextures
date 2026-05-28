package validation

import (
	"testing"
	"time"
)

// AC-4: lateness is determined from UTC instants, not display time zones.
func TestDueAtLateness_UTCComparison(t *testing.T) {
	due := time.Date(2026, 4, 15, 23, 59, 0, 0, time.UTC)
	submitted := time.Date(2026, 4, 15, 23, 58, 0, 0, time.UTC)
	if !submitted.Before(due) {
		t.Fatal("submission one minute before UTC due should be on-time")
	}
	late := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	if !late.After(due) {
		t.Fatal("submission after UTC due should be late")
	}
}
