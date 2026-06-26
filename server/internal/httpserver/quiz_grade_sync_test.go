package httpserver

import (
	"testing"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
)

func TestCanvasQuizQuestionScoresFromResponses_mapsByQuestionID(t *testing.T) {
	responses := []quizattempts.ResponseRow{
		{QuestionIndex: 0, QuestionID: "canvas-101", PointsAwarded: 4},
		{QuestionIndex: 1, QuestionID: "canvas-202", PointsAwarded: 2.5},
	}
	got := canvasQuizQuestionScoresFromResponses(responses, nil)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if score, ok := coerceCanvasJSONNumber(got["101"]["score"]); !ok || score != 4 {
		t.Fatalf("q101 score=%v", got["101"]["score"])
	}
	if score, ok := coerceCanvasJSONNumber(got["202"]["score"]); !ok || score != 2.5 {
		t.Fatalf("q202 score=%v", got["202"]["score"])
	}
}

func TestCanvasQuizQuestionScoresFromResponses_fallsBackToQuizIndex(t *testing.T) {
	questions := []coursemodulequiz.QuizQuestion{
		{ID: "canvas-55", Points: 1},
		{ID: "canvas-66", Points: 1},
	}
	responses := []quizattempts.ResponseRow{
		{QuestionIndex: 1, QuestionID: "", PointsAwarded: 1},
	}
	got := canvasQuizQuestionScoresFromResponses(responses, questions)
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	if score, ok := coerceCanvasJSONNumber(got["66"]["score"]); !ok || score != 1 {
		t.Fatalf("q66 score=%v", got["66"]["score"])
	}
}

