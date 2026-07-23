package quizgenerationai

import (
	"context"
	"testing"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

func TestHealth(t *testing.T) {
	s := New()
	if s.Name != "quizgenerationai" {
		t.Fatalf("Name = %q", s.Name)
	}
	got, err := s.Health(context.Background())
	if err != nil || got != "quizgenerationai:ok" {
		t.Fatalf("Health = %q, %v", got, err)
	}
}

func TestNormalizeQuestions(t *testing.T) {
	idx := uint(1)
	in := []coursemodulequiz.QuizQuestion{
		{Prompt: "  ", QuestionType: "multiple_choice"},
		{Prompt: "What is 2+2?", QuestionType: "bogus", CorrectChoiceIndex: &idx, Choices: []string{"3", "4"}},
		{Prompt: "True or false?", QuestionType: "true_false"},
	}
	out := normalizeQuestions(in)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].QuestionType != "short_answer" {
		t.Fatalf("invalid type not remapped: %q", out[0].QuestionType)
	}
	if out[0].ID == "" || out[0].Points != 1 {
		t.Fatalf("defaults missing: %+v", out[0])
	}
	if out[0].CorrectChoiceIndex == nil || *out[0].CorrectChoiceIndex != 1 {
		t.Fatalf("correct index = %v", out[0].CorrectChoiceIndex)
	}
	if len(out[1].Choices) != 2 || out[1].Choices[0] != "True" {
		t.Fatalf("true_false choices = %#v", out[1].Choices)
	}
}

func TestStripJSONFences(t *testing.T) {
	got := stripJSONFences("```json\n{\"questions\":[]}\n```")
	if got != `{"questions":[]}` {
		t.Fatalf("got %q", got)
	}
}
