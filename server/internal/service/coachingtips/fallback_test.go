package coachingtips

import (
	"strings"
	"testing"
)

func TestPickFallbackDeterministic(t *testing.T) {
	a := PickFallback("user:2026-05-19")
	b := PickFallback("user:2026-05-19")
	if a != b {
		t.Fatalf("expected same tip, got %q vs %q", a, b)
	}
	c := PickFallback("other:2026-05-19")
	if len(c) == 0 {
		t.Fatal("expected non-empty tip")
	}
}

func TestAggregateContextStringNoPII(t *testing.T) {
	score := 72.5
	ctx := AggregateContext{
		AvgDailyTimeMinutes: 45,
		LoginsLast7Days:     5,
		AvgQuizScore:        &score,
		ScoreTrend:          "improving",
		TopStudyWeekdays:    []string{"Tue", "Thu"},
	}
	s := ctx.String()
	if strings.Contains(s, "@") {
		t.Fatalf("context should be aggregate only: %s", s)
	}
}
