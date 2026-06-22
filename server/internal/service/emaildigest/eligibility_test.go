package emaildigest

import (
	"testing"
	"time"
)

func TestShouldSendDigest_differentTimezones(t *testing.T) {
	utcSeven := time.Date(2026, 6, 22, 7, 1, 0, 0, time.UTC)
	utc := "UTC"
	if !ShouldSendDigest(utcSeven, &utc) {
		t.Fatal("expected digest window at 07:01 UTC for UTC user")
	}

	pacific, _ := time.LoadLocation("America/Los_Angeles")
	// 2026-06-22 07:01 PDT = 2026-06-22 14:01 UTC
	pacificSeven := time.Date(2026, 6, 22, 14, 1, 0, 0, time.UTC)
	pacificTZ := "America/Los_Angeles"
	if !ShouldSendDigest(pacificSeven, &pacificTZ) {
		t.Fatal("expected digest window at 07:01 Pacific for Pacific user")
	}

	// Same instant is outside window for UTC user.
	if ShouldSendDigest(pacificSeven, &utc) {
		t.Fatal("Pacific 07:01 should not trigger digest for UTC user")
	}

	// Outside the 5-minute window.
	latePacific := time.Date(2026, 6, 22, 7, 10, 0, 0, pacific)
	if ShouldSendDigest(latePacific.In(time.UTC), &pacificTZ) {
		t.Fatal("expected outside digest window after 5 minutes")
	}
}

func TestShouldSendDigest_nilTimezoneFallsBackToUTC(t *testing.T) {
	utcSeven := time.Date(2026, 6, 22, 7, 1, 0, 0, time.UTC)
	if !ShouldSendDigest(utcSeven, nil) {
		t.Fatal("nil timezone should fall back to UTC")
	}
}

func TestLocalDayStartUTC(t *testing.T) {
	tokyo := "Asia/Tokyo"
	// 2026-06-22 20:00 JST = 2026-06-22 11:00 UTC
	now := time.Date(2026, 6, 22, 11, 0, 0, 0, time.UTC)
	start := LocalDayStartUTC(now, &tokyo)
	loc, _ := time.LoadLocation(tokyo)
	want := time.Date(2026, 6, 22, 0, 0, 0, 0, loc)
	if !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start, want)
	}
}