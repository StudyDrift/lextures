package quizattemptgrading

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

func uintPtr(u uint) *uint { return &u }

func TestGradeResponseItem_multipleChoice(t *testing.T) {
	q := coursemodulequiz.QuizQuestion{
		ID:                 "q1",
		Prompt:             "Pick one",
		QuestionType:       "multiple_choice",
		Points:             2,
		CorrectChoiceIndex: uintPtr(1),
	}
	item := coursemodulequiz.QuizQuestionResponseItem{
		QuestionID:          "q1",
		SelectedChoiceIndex: uintPtr(1),
	}
	gr := GradeResponseItem(q, item)
	if gr.PointsAwarded != 2 {
		t.Fatalf("expected 2 points, got %v", gr.PointsAwarded)
	}
	if gr.IsCorrect == nil || !*gr.IsCorrect {
		t.Fatal("expected correct")
	}
}

func TestGradeResponseItem_numeric(t *testing.T) {
	cfg, _ := json.Marshal(map[string]any{"correct": 42.0, "toleranceAbs": 0.5})
	q := coursemodulequiz.QuizQuestion{
		ID:           "q2",
		QuestionType: "numeric",
		TypeConfig:   cfg,
		Points:       1,
	}
	v := 42.25
	item := coursemodulequiz.QuizQuestionResponseItem{QuestionID: "q2", NumericValue: &v}
	gr := GradeResponseItem(q, item)
	if gr.IsCorrect == nil || !*gr.IsCorrect {
		t.Fatal("expected numeric answer within tolerance to be correct")
	}
}

func TestGradeStaticResponses(t *testing.T) {
	questions := []coursemodulequiz.QuizQuestion{
		{ID: "a", QuestionType: "multiple_choice", Points: 1, CorrectChoiceIndex: uintPtr(0)},
		{ID: "b", QuestionType: "multiple_choice", Points: 1, CorrectChoiceIndex: uintPtr(1)},
	}
	responses := []coursemodulequiz.QuizQuestionResponseItem{
		{QuestionID: "a", SelectedChoiceIndex: uintPtr(0)},
		{QuestionID: "b", SelectedChoiceIndex: uintPtr(0)},
	}
	rows := GradeStaticResponses(questions, responses)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	earned, possible := SumGradedPoints(rows)
	if possible != 2 {
		t.Fatalf("expected possible 2, got %v", possible)
	}
	if earned != 1 {
		t.Fatalf("expected earned 1, got %v", earned)
	}
}
