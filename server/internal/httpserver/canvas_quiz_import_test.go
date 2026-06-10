package httpserver

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

func TestCanvasQuestionToQuizQuestion_MultipleChoice(t *testing.T) {
	q := map[string]any{
		"id":              float64(42),
		"question_type":   "multiple_choice_question",
		"question_text":   "<p>Pick one</p>",
		"points_possible": float64(2),
		"answers": []any{
			map[string]any{"text": "A", "weight": float64(0)},
			map[string]any{"text": "B", "weight": float64(100)},
		},
	}
	qq, ok := canvasQuestionToQuizQuestion(q)
	if !ok {
		t.Fatal("expected ok")
	}
	if qq.ID != "canvas-42" || qq.QuestionType != "multiple_choice" {
		t.Fatalf("unexpected: %+v", qq)
	}
	if len(qq.Choices) != 2 || qq.Choices[1] != "B" {
		t.Fatalf("choices: %#v", qq.Choices)
	}
	if qq.CorrectChoiceIndex == nil || *qq.CorrectChoiceIndex != 1 {
		t.Fatalf("correct index: %v", qq.CorrectChoiceIndex)
	}
	if qq.Points != 2 {
		t.Fatalf("points: %d", qq.Points)
	}
}

func TestCanvasQuestionToQuizQuestion_TrueFalseSecondAnswerCorrect(t *testing.T) {
	q := map[string]any{
		"id":              float64(7),
		"question_type":   "true_false_question",
		"question_text":   "",
		"question_name":   "TF",
		"points_possible": float64(1),
		"answers": []any{
			map[string]any{"text": "False", "weight": float64(0)},
			map[string]any{"text": "True", "weight": float64(100)},
		},
	}
	qq, ok := canvasQuestionToQuizQuestion(q)
	if !ok {
		t.Fatal("expected ok")
	}
	if qq.QuestionType != "true_false" || len(qq.Choices) != 2 {
		t.Fatalf("unexpected: %+v", qq)
	}
	if qq.CorrectChoiceIndex == nil || *qq.CorrectChoiceIndex != 0 {
		t.Fatalf("want True (index 0), got %v", qq.CorrectChoiceIndex)
	}
}

func TestCanvasMatchingPairsJSON(t *testing.T) {
	t.Run("create API fields", func(t *testing.T) {
		answers := []map[string]any{
			{"answer_match_left": "A", "answer_match_right": "1", "weight": float64(100)},
		}
		raw := canvasMatchingPairsJSON(nil, answers)
		var wrap struct {
			Pairs []struct {
				LeftID  string `json:"leftId"`
				RightID string `json:"rightId"`
				Left    string `json:"left"`
				Right   string `json:"right"`
			} `json:"pairs"`
		}
		if err := json.Unmarshal(raw, &wrap); err != nil {
			t.Fatal(err)
		}
		if len(wrap.Pairs) != 1 || wrap.Pairs[0].LeftID != "l0" || wrap.Pairs[0].RightID != "r0" {
			t.Fatalf("%+v", wrap)
		}
		if wrap.Pairs[0].Left != "A" || wrap.Pairs[0].Right != "1" {
			t.Fatalf("unexpected pair text: %+v", wrap.Pairs[0])
		}
	})

	t.Run("stored Canvas question_data fields", func(t *testing.T) {
		q := map[string]any{
			"matches": []any{
				map[string]any{"match_id": float64(10), "text": "Utah"},
				map[string]any{"match_id": float64(11), "text": "Nevada"},
			},
		}
		answers := []map[string]any{
			{"id": float64(1), "left": "Salt Lake City", "right": "Utah", "match_id": float64(10)},
			{"id": float64(2), "text": "Las Vegas", "match_id": float64(11)},
		}
		raw := canvasMatchingPairsJSON(q, answers)
		var wrap struct {
			Pairs []struct {
				Left  string `json:"left"`
				Right string `json:"right"`
			} `json:"pairs"`
		}
		if err := json.Unmarshal(raw, &wrap); err != nil {
			t.Fatal(err)
		}
		if len(wrap.Pairs) != 2 {
			t.Fatalf("expected 2 pairs, got %+v", wrap)
		}
		if wrap.Pairs[0].Left != "Salt Lake City" || wrap.Pairs[0].Right != "Utah" {
			t.Fatalf("pair 0: %+v", wrap.Pairs[0])
		}
		if wrap.Pairs[1].Left != "Las Vegas" || wrap.Pairs[1].Right != "Nevada" {
			t.Fatalf("pair 1: %+v", wrap.Pairs[1])
		}
	})
}

func TestCanvasQuestionToQuizQuestion_Matching(t *testing.T) {
	q := map[string]any{
		"id":            float64(99),
		"question_type": "matching_question",
		"question_text": "<p>Match cities to states</p>",
		"points_possible": float64(2),
		"matches": []any{
			map[string]any{"match_id": float64(10), "text": "Utah"},
		},
		"answers": []any{
			map[string]any{"id": float64(1), "left": "Salt Lake City", "right": "Utah", "match_id": float64(10)},
		},
	}
	qq, ok := canvasQuestionToQuizQuestion(q)
	if !ok {
		t.Fatal("expected ok")
	}
	if qq.QuestionType != "matching" {
		t.Fatalf("unexpected type: %s", qq.QuestionType)
	}
	var cfg struct {
		Pairs []struct {
			Left  string `json:"left"`
			Right string `json:"right"`
		} `json:"pairs"`
	}
	if err := json.Unmarshal(qq.TypeConfig, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Pairs) != 1 || cfg.Pairs[0].Left != "Salt Lake City" || cfg.Pairs[0].Right != "Utah" {
		t.Fatalf("unexpected typeConfig: %+v", cfg)
	}
}

func TestCanvasQuestionIDFromLocalID(t *testing.T) {
	id, ok := canvasQuestionIDFromLocalID("canvas-42")
	if !ok || id != 42 {
		t.Fatalf("got %d ok=%v", id, ok)
	}
	if _, ok := canvasQuestionIDFromLocalID("local-1"); ok {
		t.Fatal("expected false for non-canvas id")
	}
}

func TestCanvasParseSubmissionData(t *testing.T) {
	raw := []any{
		map[string]any{
			"question_id": float64(10),
			"answer":      float64(55),
			"points":      float64(1),
			"correct":     "true",
		},
	}
	answers := canvasParseSubmissionData(raw)
	if len(answers) != 1 || answers[0].CanvasQuestionID != 10 {
		t.Fatalf("unexpected: %+v", answers)
	}
	if answers[0].Points == nil || *answers[0].Points != 1 {
		t.Fatalf("points: %+v", answers[0].Points)
	}
}

func TestCanvasUnwrapQuizSubmission(t *testing.T) {
	wrapped := map[string]any{
		"quiz_submissions": []any{
			map[string]any{
				"id": float64(99),
				"submission_data": []any{
					map[string]any{
						"question_id": float64(5),
						"points":      float64(2),
						"correct":     true,
					},
				},
			},
		},
	}
	unwrapped := canvasUnwrapQuizSubmission(wrapped)
	if int64At(unwrapped, "id") != 99 {
		t.Fatalf("expected id 99, got %+v", unwrapped)
	}
	answers := canvasParseSubmissionData(unwrapped["submission_data"])
	if len(answers) != 1 || answers[0].CanvasQuestionID != 5 || answers[0].Points == nil || *answers[0].Points != 2 {
		t.Fatalf("unexpected answers: %+v", answers)
	}
}

func TestCanvasParseSubmissionDataMap(t *testing.T) {
	raw := map[string]any{
		"101": map[string]any{"score": float64(1), "correct": "true"},
	}
	answers := canvasParseSubmissionData(raw)
	if len(answers) != 1 || answers[0].CanvasQuestionID != 101 {
		t.Fatalf("unexpected: %+v", answers)
	}
}

func TestCanvasGradeImportedMultipleChoice(t *testing.T) {
	correct := uint(1)
	q := coursemodulequiz.QuizQuestion{
		ID:                 "canvas-7",
		QuestionType:       "multiple_choice",
		CorrectChoiceIndex: &correct,
		Points:             2,
	}
	choiceMaps := map[int64]int{100: 0, 200: 1}
	answer := canvasQuizSubmissionAnswer{
		CanvasQuestionID: 7,
		Answer:           float64(200),
	}
	_, isCorrect, pts, max := canvasGradeImportedQuestion(q, answer, choiceMaps, nil)
	if isCorrect == nil || !*isCorrect {
		t.Fatalf("expected correct, got %v", isCorrect)
	}
	if pts != 2 || max != 2 {
		t.Fatalf("points earned=%v max=%v", pts, max)
	}
}
