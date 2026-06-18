package onboarding

import "testing"

func TestEffectiveLevel_DiagnosticOverridesSelfAssessment(t *testing.T) {
	score := 75.0
	got := EffectiveLevel("beginner", &score, false)
	if got != "advanced" {
		t.Fatalf("want advanced got %q", got)
	}
}

func TestEffectiveLevel_SelfAssessmentWhenDiagnosticSkipped(t *testing.T) {
	got := EffectiveLevel("intermediate", nil, true)
	if got != "intermediate" {
		t.Fatalf("want intermediate got %q", got)
	}
}

func TestScoreDiagnostic(t *testing.T) {
	answers := map[string]int{
		"py1": 1,
		"py2": 1,
		"py3": 1,
		"py4": 2,
		"py5": 1,
	}
	score := ScoreDiagnostic("python", answers)
	if score != 100 {
		t.Fatalf("want 100 got %v", score)
	}
}

func TestLevelKeywordsFor(t *testing.T) {
	kw := levelKeywordsFor("beginner")
	if len(kw) == 0 {
		t.Fatal("expected beginner keywords")
	}
}
