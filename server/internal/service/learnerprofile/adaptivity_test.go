package learnerprofile

import (
	"testing"
	"time"
)

func TestApplyRecommendations_InterestAndGrowth(t *testing.T) {
	ctx := AdaptiveContext{
		Active:      true,
		EvalCohort:  "personalised",
		Interests:   []string{"algebra"},
		GrowthAreas: []string{"fractions"},
	}
	items := []RecommendationItem{
		{Title: "Intro to algebra", Score: 0.7, Surface: "continue"},
		{Title: "Practice fractions", Score: 0.68, Surface: "strengthen"},
		{Title: "Unrelated topic", Score: 0.5, Surface: "continue"},
	}
	out := ApplyRecommendations(ctx, items)
	if out[0].Title != "Intro to algebra" {
		t.Fatalf("expected interest-boosted item first, got %q", out[0].Title)
	}
	if out[0].Rationale == nil {
		t.Fatal("expected rationale on top item")
	}
}

func TestApplyRecommendations_SuppressedWhenInactive(t *testing.T) {
	ctx := AdaptiveContext{Active: false}
	items := []RecommendationItem{{Title: "algebra", Score: 0.5}}
	out := ApplyRecommendations(ctx, items)
	if out[0].Rationale != nil {
		t.Fatal("expected no rationale when inactive")
	}
}

func TestApplyReviewQueue_NeedsReviewFirst(t *testing.T) {
	ctx := AdaptiveContext{
		Active:      true,
		EvalCohort:  "personalised",
		NeedsReview: []string{"photosynthesis"},
		PeakWindows: []PeakWindowFacet{{Dow: "weekday", HourBucket: "18-21", Share: 0.4}},
	}
	now := time.Date(2026, 7, 7, 19, 0, 0, 0, time.UTC) // Monday evening UTC
	items := []ReviewQueueItem{
		{StateID: "a", Stem: "What is mitosis?", NextReviewAt: "2026-07-07T10:00:00Z"},
		{StateID: "b", Stem: "Explain photosynthesis", NextReviewAt: "2026-07-07T11:00:00Z"},
	}
	out := ApplyReviewQueue(ctx, items, now)
	if out[0].StateID != "b" {
		t.Fatalf("expected needs-review concept first, got %q", out[0].StateID)
	}
	if out[0].Rationale == nil || out[0].Rationale.InsightKey != "needs_review" {
		t.Fatalf("rationale=%v", out[0].Rationale)
	}
}

func TestAdaptiveContext_Usable_ControlCohort(t *testing.T) {
	ctx := AdaptiveContext{Active: true, EvalCohort: "control"}
	if ctx.Usable(true) {
		t.Fatal("control cohort should suppress personalisation")
	}
	if !ctx.Usable(false) {
		t.Fatal("cohort gate off should allow active context")
	}
}

func TestClassifyContentModality(t *testing.T) {
	if got := classifyContentModality("# Hello\n\nPlain text."); got != "reading" {
		t.Fatalf("got %q", got)
	}
	if got := classifyContentModality("Watch https://www.youtube.com/watch?v=abc"); got != "video" {
		t.Fatalf("got %q", got)
	}
}

func TestTutorScaffoldingPrompt_EarlyReliance(t *testing.T) {
	p := TutorScaffoldingPrompt("early-reliance")
	if p == "" || !containsSubstr(p, "nudge") {
		t.Fatalf("prompt=%q", p)
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexSubstr(s, sub))
}

func indexSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}