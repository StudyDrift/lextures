package quizgameai

import (
	"testing"

	"github.com/lextures/lextures/server/internal/repos/quizgame"
)

func TestParseModelJSON_Valid(t *testing.T) {
	raw := `{
		"questions": [
			{
				"questionType": "mc_single",
				"prompt": "What is 2+2?",
				"options": [
					{"id": "a", "text": "3", "isCorrect": false},
					{"id": "b", "text": "4", "isCorrect": true},
					{"id": "c", "text": "5", "isCorrect": false},
					{"id": "d", "text": "22", "isCorrect": false}
				],
				"timeLimitSeconds": 20,
				"explanation": "Basic arithmetic",
				"confidence": 0.9
			}
		],
		"suggestedSubject": "Math",
		"suggestedGradeBand": "3-5"
	}`
	payload, err := ParseModelJSON(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(payload.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(payload.Questions))
	}
	if payload.SuggestedSubject != "Math" {
		t.Fatalf("subject: %q", payload.SuggestedSubject)
	}
}

func TestParseModelJSON_Malformed(t *testing.T) {
	if _, err := ParseModelJSON("not json"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateAndFilter_DropsInvalidKeepsValid(t *testing.T) {
	confHigh := 0.9
	confLow := 0.2
	drafts := []DraftQuestion{
		{
			QuestionType: quizgame.QTypeMCSingle,
			Prompt:       "Capital of France?",
			Options: []quizgame.Option{
				{ID: "a", Text: "Paris", IsCorrect: true},
				{ID: "b", Text: "Lyon", IsCorrect: false},
				{ID: "c", Text: "Nice", IsCorrect: false},
				{ID: "d", Text: "Lille", IsCorrect: false},
			},
			TimeLimitSeconds: 20,
			Confidence:       &confHigh,
		},
		{
			QuestionType:     quizgame.QTypeMCSingle,
			Prompt:           "",
			Options:          []quizgame.Option{{ID: "a", Text: "x", IsCorrect: true}},
			TimeLimitSeconds: 20,
			Confidence:       &confLow,
		},
		{
			QuestionType: "essay",
			Prompt:       "Write an essay",
			Confidence:   &confHigh,
		},
	}
	res := ValidateAndFilter(drafts, []string{quizgame.QTypeMCSingle, quizgame.QTypeTrueFalse}, true, "11111111-1111-1111-1111-111111111111")
	if len(res.Inputs) != 1 {
		t.Fatalf("expected 1 valid, got %d (dropped=%d)", len(res.Inputs), res.Dropped)
	}
	if res.Dropped != 2 {
		t.Fatalf("expected 2 dropped, got %d", res.Dropped)
	}
	in := res.Inputs[0]
	if in.Source != quizgame.QuestionSourceAIGenerated {
		t.Fatalf("source: %q", in.Source)
	}
	if in.NeedsReview == nil || !*in.NeedsReview {
		t.Fatal("expected needs_review")
	}
}

func TestRedactSource_StripsEmail(t *testing.T) {
	out := RedactSource(SourceMaterial{
		Topic:   "email alice@school.edu about photosynthesis",
		Passage: "Contact bob@example.com for help",
	})
	if out.Topic == "email alice@school.edu about photosynthesis" {
		t.Fatalf("topic not redacted: %q", out.Topic)
	}
	if out.Passage == "Contact bob@example.com for help" {
		t.Fatalf("passage not redacted: %q", out.Passage)
	}
}

func TestNormalizeGenerationParams_Defaults(t *testing.T) {
	p := quizgame.GenerationParams{}
	if err := quizgame.NormalizeGenerationParams(&p); err != nil {
		t.Fatal(err)
	}
	if p.Count != 5 {
		t.Fatalf("count: %d", p.Count)
	}
	if p.Language != "en" {
		t.Fatalf("lang: %q", p.Language)
	}
}
