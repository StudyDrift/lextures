package studyreminders

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestShouldSendDaily(t *testing.T) {
	tz := "America/New_York"
	reminder := time.Date(0, 1, 1, 19, 0, 0, 0, time.UTC)
	localDate := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	loc, _ := time.LoadLocation(tz)
	localNow := time.Date(2026, 6, 18, 19, 2, 0, 0, loc)

	if ShouldSendDaily(localNow, localDate, reminder, &tz, true) {
		t.Fatal("studied today should skip daily reminder")
	}
	if !ShouldSendDaily(localNow, localDate, reminder, &tz, false) {
		t.Fatal("expected daily reminder window")
	}
	late := time.Date(2026, 6, 18, 19, 10, 0, 0, loc)
	if ShouldSendDaily(late, localDate, reminder, &tz, false) {
		t.Fatal("outside window should not send")
	}
}

func TestShouldSendStreakAtRisk(t *testing.T) {
	tz := "UTC"
	reminder := time.Date(0, 1, 1, 21, 0, 0, 0, time.UTC)
	localDate := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	atRiskTime := time.Date(2026, 6, 18, 19, 1, 0, 0, time.UTC)

	if !ShouldSendStreakAtRisk(atRiskTime, localDate, reminder, &tz, false, 5) {
		t.Fatal("expected streak-at-risk window 2h before 9pm")
	}
	if ShouldSendStreakAtRisk(atRiskTime, localDate, reminder, &tz, true, 5) {
		t.Fatal("studied today should skip streak-at-risk")
	}
}

func TestIdempotencyKeyStable(t *testing.T) {
	uid := mustParseUUID("11111111-1111-1111-1111-111111111111")
	d := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	k1 := IdempotencyKey(uid, d, "daily", "email")
	k2 := IdempotencyKey(uid, d, "daily", "email")
	if k1 != k2 {
		t.Fatalf("keys differ: %q vs %q", k1, k2)
	}
}

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
