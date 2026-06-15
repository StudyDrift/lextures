package advising

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCacheExpired(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	fetched := now.Add(-3 * time.Hour)
	if CacheExpired(fetched, now) {
		t.Fatal("3h old cache should not be expired")
	}
	fetched = now.Add(-4 * time.Hour)
	if !CacheExpired(fetched, now) {
		t.Fatal("4h old cache should be expired")
	}
}

func TestFulfillsRequirements(t *testing.T) {
	summary := &DegreeProgressSummary{
		CourseRequirements: map[string][]string{
			"MATH201": {"Core Mathematics"},
		},
	}
	got := FulfillsRequirements(summary, "MATH201")
	if len(got) != 1 || got[0] != "Core Mathematics" {
		t.Fatalf("got %v", got)
	}
	if FulfillsRequirements(summary, "UNKNOWN") != nil {
		t.Fatal("unknown course should return nil")
	}
}

func TestStubSummaryDeterministic(t *testing.T) {
	uid := uuid.MustParse("00000000-0000-0000-0000-000000000042")
	a := stubSummary(uid, ProviderDegreeWorks)
	b := stubSummary(uid, ProviderDegreeWorks)
	if a.CompletionPercent != b.CompletionPercent {
		t.Fatal("stub should be deterministic for same user")
	}
}
