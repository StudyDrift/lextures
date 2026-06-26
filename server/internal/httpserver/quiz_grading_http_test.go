package httpserver

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
)

func TestBuildQuizGradingQuestions_includesAllResponsesAndQuizDefinition(t *testing.T) {
	qid := "canvas-1"
	responses := []quizattempts.ResponseRow{
		{
			QuestionIndex:  0,
			QuestionID:     "canvas-10",
			QuestionType:   "multiple_choice",
			PromptSnapshot: "Pick one",
			ResponseJSON:   json.RawMessage(`{"selectedChoiceIndex":0}`),
			PointsAwarded:  12,
			MaxPoints:      12,
		},
		{
			QuestionIndex:  1,
			QuestionID:     "canvas-11",
			QuestionType:   "essay",
			PromptSnapshot: "Explain your work",
			ResponseJSON:   json.RawMessage(`{}`),
			MaxPoints:      12,
		},
	}
	quizQuestions := []coursemodulequiz.QuizQuestion{
		{ID: "canvas-10", Prompt: "Pick one", QuestionType: "multiple_choice", Points: 12},
		{ID: "canvas-11", Prompt: "Explain your work", QuestionType: "essay", Points: 12},
		{ID: qid, Prompt: "Skipped question", QuestionType: "true_false", Points: 5},
	}

	got := buildQuizGradingQuestions(responses, quizQuestions)
	if len(got) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(got))
	}
	if got[0].QuestionType != "multiple_choice" || got[0].PointsAwarded == nil || *got[0].PointsAwarded != 12 {
		t.Fatalf("graded MC missing: %+v", got[0])
	}
	if got[1].NeedsGrading != true {
		t.Fatalf("essay should still need grading: %+v", got[1])
	}
	if got[2].QuestionType != "true_false" || got[2].PromptSnapshot == nil || *got[2].PromptSnapshot != "Skipped question" {
		t.Fatalf("quiz-definition-only question missing: %+v", got[2])
	}
}