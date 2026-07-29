package inlinequestionsai

import (
	"context"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"
)

func TestServiceHealth(t *testing.T) {
	s := New()
	if s.Name != "inlinequestionsai" {
		t.Fatalf("name: %q", s.Name)
	}
	got, err := s.Health(context.Background())
	if err != nil || got != "inlinequestionsai:ok" {
		t.Fatalf("health: got %q err %v", got, err)
	}
}

func TestParseModelJSON_NormalizesTypes(t *testing.T) {
	raw := `{
  "label": "Check understanding",
  "questions": [
    {
      "type": "single",
      "prompt": "What is 2+2?",
      "options": [
        {"text": "3", "correct": false, "feedback": "Too low"},
        {"text": "4", "correct": true},
        {"text": "5", "correct": false}
      ],
      "explanation": "Basic arithmetic."
    },
    {
      "type": "true_false",
      "prompt": "The sky is blue.",
      "options": [
        {"text": "True", "correct": true},
        {"text": "False", "correct": false}
      ]
    },
    {
      "type": "short_text",
      "prompt": "Capital of France?",
      "acceptedAnswers": ["Paris", "paris"]
    },
    {
      "type": "numeric",
      "prompt": "Half of 10?",
      "correctValue": 5,
      "tolerance": {"kind": "absolute", "value": 0.1},
      "unit": ""
    }
  ]
}`
	out, err := parseModelJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Label != "Check understanding" {
		t.Fatalf("label: %q", out.Label)
	}
	// Cap at MaxQuestions (3).
	if len(out.Questions) != 3 {
		t.Fatalf("questions len: %d", len(out.Questions))
	}
	if out.Questions[0].Type != inline_questions.TypeSingle {
		t.Fatalf("q0 type: %s", out.Questions[0].Type)
	}
	if out.Questions[0].ID == "" || len(out.Questions[0].Options) != 3 {
		t.Fatalf("q0 options: %+v", out.Questions[0])
	}
	if out.Questions[1].Type != inline_questions.TypeTrueFalse {
		t.Fatalf("q1 type: %s", out.Questions[1].Type)
	}
	if out.Questions[1].Options[0].ID != "true" || out.Questions[1].Options[1].ID != "false" {
		t.Fatalf("tf options: %+v", out.Questions[1].Options)
	}
	if out.Questions[2].Type != inline_questions.TypeShortText {
		t.Fatalf("q2 type: %s", out.Questions[2].Type)
	}
	if len(out.Questions[2].AcceptedAnswers) != 2 {
		t.Fatalf("accepted: %+v", out.Questions[2].AcceptedAnswers)
	}
}

func TestParseModelJSON_EmptyQuestions(t *testing.T) {
	out, err := parseModelJSON(`{"questions":[]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Questions) != 0 {
		t.Fatalf("expected empty, got %d", len(out.Questions))
	}
}

func TestParseModelJSON_RejectsEmptyPrompt(t *testing.T) {
	out, err := parseModelJSON(`{"questions":[{"type":"single","prompt":"","options":[{"text":"A","correct":true},{"text":"B","correct":false}]}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Questions) != 0 {
		t.Fatalf("expected empty after filter, got %d", len(out.Questions))
	}
}
