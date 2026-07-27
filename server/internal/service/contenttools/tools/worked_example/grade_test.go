package worked_example

import "testing"

func TestGradeExpressionEquivalent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Variables = []string{"x"}
	step := Step{
		ID:   "s1",
		Text: "expand",
		Blank: &Blank{
			Type:     BlankExpression,
			Expected: "3(x+2)",
		},
	}
	g := GradeStep(cfg, step, "3x + 6")
	if g.Result != ResultCorrect {
		t.Fatalf("got %s, want correct", g.Result)
	}
}

func TestGradeExpressionNeedsReview(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Variables = []string{"x"}
	step := Step{
		ID: "s1",
		Blank: &Blank{
			Type:     BlankExpression,
			Expected: "sin(x)", // unsupported → undecidable on expected side too
		},
	}
	g := GradeStep(cfg, step, "cos(x)")
	if g.Result != ResultNeedsReview {
		t.Fatalf("got %s, want needs_review", g.Result)
	}
}

func TestGradeNumericLocale(t *testing.T) {
	cfg := DefaultConfig()
	step := Step{
		ID: "s1",
		Blank: &Blank{
			Type:     BlankNumeric,
			Expected: 3.14,
			Tolerance: &Tolerance{Kind: ToleranceAbsolute, Value: 0.01},
		},
	}
	g := GradeStep(cfg, step, "3,14")
	if g.Result != ResultCorrect {
		t.Fatalf("got %s for de-DE decimal, want correct", g.Result)
	}
}

func TestGradeChoiceAndText(t *testing.T) {
	cfg := DefaultConfig()
	choice := Step{
		ID: "c1",
		Blank: &Blank{
			Type: BlankChoice,
			Options: []ChoiceOption{
				{ID: "a", Text: "Distribute"},
				{ID: "b", Text: "Factor"},
			},
			CorrectOptionID: "a",
		},
	}
	if GradeStep(cfg, choice, "a").Result != ResultCorrect {
		t.Fatal("choice correct failed")
	}
	if GradeStep(cfg, choice, "b").Result != ResultIncorrect {
		t.Fatal("choice incorrect failed")
	}
	text := Step{
		ID: "t1",
		Blank: &Blank{
			Type:            BlankText,
			AcceptedAnswers: []string{"distributive property"},
		},
	}
	if GradeStep(cfg, text, "Distributive Property").Result != ResultCorrect {
		t.Fatal("text correct failed")
	}
}

func TestBlankingProgressiveDeterministic(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BlankPolicy = BlankProgressive
	cfg.Steps = []Step{
		{ID: "s1", Text: "1", Blank: &Blank{Type: BlankNumeric, Expected: 1}},
		{ID: "s2", Text: "2", Blank: &Blank{Type: BlankNumeric, Expected: 2}},
		{ID: "s3", Text: "3", Blank: &Blank{Type: BlankNumeric, Expected: 3}},
		{ID: "s4", Text: "4", Blank: &Blank{Type: BlankNumeric, Expected: 4}},
		{ID: "s5", Text: "5", Blank: &Blank{Type: BlankNumeric, Expected: 5}},
	}
	a := ResolveBlanked(cfg, 42)
	b := ResolveBlanked(cfg, 42)
	if len(a) != len(b) {
		t.Fatalf("non-deterministic length")
	}
	for k, v := range a {
		if b[k] != v {
			t.Fatalf("non-deterministic blanking for %s", k)
		}
	}
	if len(a) == 0 {
		t.Fatal("expected at least one blanked step")
	}
	// Later steps should tend to be blanked more often than early ones across seeds.
	early := 0
	late := 0
	for seed := uint64(0); seed < 50; seed++ {
		m := ResolveBlanked(cfg, seed)
		if m["s1"] {
			early++
		}
		if m["s5"] {
			late++
		}
	}
	if late < early {
		t.Fatalf("expected later steps blanked at least as often (early=%d late=%d)", early, late)
	}
}

func TestComputeScoreIgnoresRevealed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PracticeOnly = false
	cfg.Steps = []Step{
		{ID: "s1", Text: "1", Blank: &Blank{Type: BlankNumeric, Expected: 1}},
		{ID: "s2", Text: "2", Blank: &Blank{Type: BlankNumeric, Expected: 2}},
	}
	blanked := map[string]bool{"s1": true, "s2": true}
	st := EmptyState()
	st.Steps["s1"] = StepProgress{
		Attempts: []Attempt{{Value: "1", Result: ResultCorrect, At: "t"}},
	}
	st.Steps["s2"] = StepProgress{
		Attempts: []Attempt{{Value: "9", Result: ResultIncorrect, At: "t"}},
		Revealed: true,
	}
	raw, max := ComputeScore(cfg, st, blanked)
	if max != 2 || raw != 1 {
		t.Fatalf("score raw=%v max=%v", raw, max)
	}
}

func TestSequentialGating(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ShowAllSteps = false
	cfg.Steps = []Step{
		{ID: "s1", Text: "1", Blank: &Blank{Type: BlankNumeric, Expected: 1}},
		{ID: "s2", Text: "2", Blank: &Blank{Type: BlankNumeric, Expected: 2}},
	}
	blanked := map[string]bool{"s1": true, "s2": true}
	st := EmptyState()
	if !StepUnlocked(cfg, st, "s1", blanked) {
		t.Fatal("s1 should be unlocked")
	}
	if StepUnlocked(cfg, st, "s2", blanked) {
		t.Fatal("s2 should be locked")
	}
	st.Steps["s1"] = StepProgress{
		Attempts:    []Attempt{{Value: "1", Result: ResultCorrect, At: "t"}},
		CompletedAt: "t",
	}
	if !StepUnlocked(cfg, st, "s2", blanked) {
		t.Fatal("s2 should unlock after s1")
	}
}

func TestVerifyExpected(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Variables = []string{"x"}
	step := Step{
		ID: "s1",
		Blank: &Blank{
			Type:     BlankExpression,
			Expected: "3x+6",
		},
	}
	if VerifyExpected(cfg, step).Result != ResultCorrect {
		t.Fatal("verify should accept author's expected")
	}
}
