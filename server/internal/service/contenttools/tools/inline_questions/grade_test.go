package inline_questions

import "testing"

func TestGradeSingle(t *testing.T) {
	q := Question{
		ID:   "q1",
		Type: TypeSingle,
		Options: []Option{
			{ID: "a", Text: "A", Correct: false, Feedback: "Nope"},
			{ID: "b", Text: "B", Correct: true, Feedback: "Yes"},
		},
		Explanation: "Because B",
		Points:      2,
	}
	got := GradeQuestion(q, "b")
	if !got.Correct || got.PointsAwarded != 2 || got.Feedback != "Yes" {
		t.Fatalf("correct case: %+v", got)
	}
	got = GradeQuestion(q, "a")
	if got.Correct || got.Feedback != "Nope" || got.PointsAwarded != 0 {
		t.Fatalf("wrong case: %+v", got)
	}
}

func TestGradeMultiStrictAndPartial(t *testing.T) {
	q := Question{
		ID:   "q1",
		Type: TypeMulti,
		Options: []Option{
			{ID: "a", Correct: true},
			{ID: "b", Correct: true},
			{ID: "c", Correct: false},
		},
		Points: 2,
	}
	got := GradeQuestion(q, []string{"a", "b"})
	if !got.Correct || got.PointsAwarded != 2 {
		t.Fatalf("strict all correct: %+v", got)
	}
	got = GradeQuestion(q, []string{"a"})
	if got.Correct || got.PointsAwarded != 0 {
		t.Fatalf("strict partial should be zero: %+v", got)
	}
	q.PartialCredit = true
	got = GradeQuestion(q, []string{"a"})
	if got.Correct || got.PointsAwarded != 1 {
		t.Fatalf("partial half: %+v", got)
	}
}

func TestGradeShortTextNormalisation(t *testing.T) {
	q := Question{
		ID:              "q1",
		Type:            TypeShortText,
		AcceptedAnswers: []string{"photosynthesis"},
	}
	if !GradeQuestion(q, " Photosynthesis ").Correct {
		t.Fatal("default trim+case should match")
	}
	if GradeQuestion(q, "photo-synthesis").Correct {
		t.Fatal("punctuation should not match without flag")
	}
	q.NormalizePunctuation = true
	if !GradeQuestion(q, "photo-synthesis").Correct {
		t.Fatal("normalizePunctuation should match")
	}
	q.CaseSensitive = true
	q.NormalizePunctuation = false
	if GradeQuestion(q, "Photosynthesis").Correct {
		t.Fatal("caseSensitive should reject")
	}
}

func TestGradeNumericTolerance(t *testing.T) {
	cv := 3.14159
	q := Question{
		ID:           "q1",
		Type:         TypeNumeric,
		CorrectValue: &cv,
		Tolerance:    &Tolerance{Kind: ToleranceAbsolute, Value: 0.05},
	}
	if !GradeQuestion(q, 3.14).Correct {
		t.Fatal("3.14 should be within ±0.05")
	}
	if GradeQuestion(q, 3.0).Correct {
		t.Fatal("3.0 should be incorrect")
	}
	q.Tolerance = &Tolerance{Kind: ToleranceRelative, Value: 0.01}
	if !GradeQuestion(q, 3.14).Correct {
		t.Fatal("relative 1% should accept 3.14")
	}
}

func TestParseNumericLocale(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"3.14", 3.14, true},
		{"3,14", 3.14, true},
		{"1.234,56", 1234.56, true},
		{"1,234.56", 1234.56, true},
		{"", 0, false},
	}
	for _, tc := range cases {
		got, ok := ParseNumericValue(tc.in)
		if ok != tc.ok {
			t.Fatalf("%q ok=%v want %v", tc.in, ok, tc.ok)
		}
		if ok && got != tc.want {
			t.Fatalf("%q = %v want %v", tc.in, got, tc.want)
		}
	}
}

func TestScorePolicy(t *testing.T) {
	cfg := Config{
		Questions: []Question{{ID: "q1", Type: TypeSingle, Points: 1}},
		ScorePolicy: ScoreBest,
		Attempts: 3,
	}
	st := EmptyState()
	st.Answers["q1"] = QuestionAnswer{Attempts: []Attempt{
		{Value: "a", Correct: false, Points: 0},
		{Value: "b", Correct: true, Points: 1},
	}}
	raw, max := ComputeScore(cfg, st)
	if raw != 1 || max != 1 {
		t.Fatalf("best: raw=%v max=%v", raw, max)
	}
	cfg.ScorePolicy = ScoreFirst
	raw, _ = ComputeScore(cfg, st)
	if raw != 0 {
		t.Fatalf("first should be 0, got %v", raw)
	}
	cfg.ScorePolicy = ScoreLast
	raw, _ = ComputeScore(cfg, st)
	if raw != 1 {
		t.Fatalf("last should be 1, got %v", raw)
	}
}

func TestAttemptsAndSequential(t *testing.T) {
	cfg := Config{
		Questions: []Question{
			{ID: "q1", Type: TypeSingle},
			{ID: "q2", Type: TypeSingle},
		},
		Attempts:   2,
		Sequential: true,
	}
	st := EmptyState()
	if !QuestionUnlocked(cfg, st, "q1") {
		t.Fatal("q1 should be unlocked")
	}
	if QuestionUnlocked(cfg, st, "q2") {
		t.Fatal("q2 should be locked")
	}
	st.Answers["q1"] = QuestionAnswer{Attempts: []Attempt{{Value: "a", Correct: false, At: "t"}}}
	if !QuestionUnlocked(cfg, st, "q2") {
		t.Fatal("q2 should unlock after q1 attempt")
	}
	if AttemptsRemaining(cfg, st, "q1") != 1 {
		t.Fatalf("remaining=%d", AttemptsRemaining(cfg, st, "q1"))
	}
	cfg.Attempts = "unlimited"
	if AttemptsRemaining(cfg, st, "q1") != -1 {
		t.Fatal("unlimited should be -1")
	}
}

func TestShouldReveal(t *testing.T) {
	cfg := Config{Attempts: 2, RevealCorrectAfter: RevealLastAttempt}
	st := EmptyState()
	st.Answers["q1"] = QuestionAnswer{Attempts: []Attempt{{Correct: false}}}
	if ShouldReveal(cfg, st, "q1", false) {
		t.Fatal("should not reveal on first miss with last_attempt")
	}
	st.Answers["q1"] = QuestionAnswer{Attempts: []Attempt{{Correct: false}, {Correct: false}}}
	if !ShouldReveal(cfg, st, "q1", false) {
		t.Fatal("should reveal when exhausted")
	}
	cfg.RevealCorrectAfter = RevealNever
	if ShouldReveal(cfg, st, "q1", true) {
		t.Fatal("never should not reveal")
	}
}
