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

func TestCanvasCanvasUserIDFromMap_nestedUser(t *testing.T) {
	raw := map[string]any{
		"user": map[string]any{"id": float64(42)},
	}
	if got := canvasCanvasUserIDFromMap(raw); got != 42 {
		t.Fatalf("expected 42, got %d", got)
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

func TestCanvasParseSubmissionDataMap_stringAnswer(t *testing.T) {
	raw := map[string]any{
		"42": "My contributions this sprint have been: shipping the API.",
	}
	answers := canvasParseSubmissionData(raw)
	if len(answers) != 1 || answers[0].CanvasQuestionID != 42 {
		t.Fatalf("unexpected: %+v", answers)
	}
	if got := canvasAnswerAsString(answers[0].Answer); got == "" {
		t.Fatalf("expected text answer, got %+v", answers[0].Answer)
	}
}

func TestCanvasMergeSubmissionAnswers_questionRowUsesQuizQuestionID(t *testing.T) {
	merged := canvasMergeSubmissionAnswers(nil, []map[string]any{
		{
			"id":     float64(42),
			"answer": "Sprint retrospective notes",
		},
	}, nil)
	ans, ok := merged[42]
	if !ok {
		t.Fatalf("expected answer keyed by quiz question id 42, got %+v", merged)
	}
	if canvasAnswerAsString(ans.Answer) != "Sprint retrospective notes" {
		t.Fatalf("unexpected answer: %+v", ans.Answer)
	}
}

func TestCanvasQuestionToQuizQuestion_FileUpload(t *testing.T) {
	q := map[string]any{
		"id":              float64(501),
		"question_type":   "file_upload_question",
		"question_text":   "<p>List the tasks that you have been assigned:</p>",
		"points_possible": float64(10),
	}
	qq, ok := canvasQuestionToQuizQuestion(q)
	if !ok {
		t.Fatalf("expected file upload question to convert")
	}
	if qq.QuestionType != "file_upload" {
		t.Fatalf("expected file_upload type, got %q", qq.QuestionType)
	}
	if qq.Points != 10 {
		t.Fatalf("expected 10 points, got %d", qq.Points)
	}
}

func TestCanvasAttachmentIDsFromMap(t *testing.T) {
	got := canvasAttachmentIDsFromMap(map[string]any{
		"attachment_ids": []any{float64(11), "12", float64(11)},
	})
	if len(got) != 2 || got[0] != 11 || got[1] != 12 {
		t.Fatalf("expected [11 12] deduped, got %v", got)
	}
	objs := canvasAttachmentIDsFromMap(map[string]any{
		"attachments": []any{map[string]any{"id": float64(99)}},
	})
	if len(objs) != 1 || objs[0] != 99 {
		t.Fatalf("expected [99] from attachment objects, got %v", objs)
	}
	if ids := canvasAttachmentIDsFromMap(map[string]any{"text": "no files"}); len(ids) != 0 {
		t.Fatalf("expected no attachment ids, got %v", ids)
	}
}

func TestCanvasParseSubmissionData_fileUploadAttachments(t *testing.T) {
	raw := []any{
		map[string]any{
			"question_id":    float64(501),
			"attachment_ids": []any{float64(11), float64(12)},
		},
	}
	answers := canvasParseSubmissionData(raw)
	if len(answers) != 1 || answers[0].CanvasQuestionID != 501 {
		t.Fatalf("unexpected: %+v", answers)
	}
	if len(answers[0].AttachmentIDs) != 2 {
		t.Fatalf("expected 2 attachment ids, got %v", answers[0].AttachmentIDs)
	}
}

func TestCanvasInjectFilesIntoResponseJSON(t *testing.T) {
	base := json.RawMessage(`{"textAnswer":"see files"}`)
	out := canvasInjectFilesIntoResponseJSON(base, []canvasImportedQuizFile{
		{Filename: "image.png", MimeType: "image/png", ContentPath: "/c/1"},
	})
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if parsed["textAnswer"] != "see files" {
		t.Fatalf("expected textAnswer preserved, got %v", parsed["textAnswer"])
	}
	files, ok := parsed["files"].([]any)
	if !ok || len(files) != 1 {
		t.Fatalf("expected 1 file, got %v", parsed["files"])
	}
}

func TestCanvasParseSubmissionDataMap_questionPrefixKeys(t *testing.T) {
	raw := map[string]any{
		"question_42": "My contributions this sprint have been: shipping the API.",
		"question_7":  map[string]any{"answer": float64(55), "correct": "true"},
		"question_42_marked": true,
	}
	answers := canvasParseSubmissionData(raw)
	byID := make(map[int64]canvasQuizSubmissionAnswer, len(answers))
	for _, a := range answers {
		byID[a.CanvasQuestionID] = a
	}
	if got := canvasAnswerAsString(byID[42].Answer); got == "" {
		t.Fatalf("expected essay text for question 42, got %+v", byID[42])
	}
	if byID[7].Answer == nil {
		t.Fatalf("expected MC answer for question 7, got %+v", byID[7])
	}
}

func TestCanvasParseSubmissionDataSlice_gradedEssayUsesText(t *testing.T) {
	raw := []any{
		map[string]any{
			"question_id": float64(9),
			"text":        "<p>Student essay response</p>",
			"correct":     "undefined",
			"points":      float64(0),
		},
	}
	answers := canvasParseSubmissionData(raw)
	if len(answers) != 1 || answers[0].CanvasQuestionID != 9 {
		t.Fatalf("unexpected: %+v", answers)
	}
	if canvasAnswerAsString(answers[0].Answer) == "" {
		t.Fatalf("expected essay text, got %+v", answers[0].Answer)
	}
}

func TestCanvasMergeSubmissionAnswers_eventsFallback(t *testing.T) {
	merged := canvasMergeSubmissionAnswers(nil, nil, []map[string]any{
		{
			"event_type": "question_answered",
			"event_data": map[string]any{
				"question_id": float64(15),
				"answer":      "Captured from quiz log auditing",
			},
		},
	})
	ans, ok := merged[15]
	if !ok {
		t.Fatalf("expected event-backed answer, got %+v", merged)
	}
	if canvasAnswerAsString(ans.Answer) != "Captured from quiz log auditing" {
		t.Fatalf("unexpected answer: %+v", ans.Answer)
	}
}

func TestCanvasAnswerTextToMarkdown(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text passes through", in: "  10  ", want: "10"},
		{name: "empty stays empty", in: "   ", want: ""},
		{name: "html converts to markdown", in: "<p>Hello <strong>world</strong></p>", want: "Hello **world**"},
		{name: "html list converts", in: "<ul><li>a</li><li>b</li></ul>", want: "- a\n- b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canvasAnswerTextToMarkdown(tc.in); got != tc.want {
				t.Fatalf("canvasAnswerTextToMarkdown(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestCanvasResponseJSONForAnswer_essayHTMLBecomesMarkdown(t *testing.T) {
	q := coursemodulequiz.QuizQuestion{ID: "canvas-42", QuestionType: "essay", Points: 5}
	got := canvasResponseJSONForAnswer(q, "<p>Reflection on the <em>sprint</em></p>", nil)
	var decoded struct {
		TextAnswer string `json:"textAnswer"`
	}
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal response json: %v", err)
	}
	if decoded.TextAnswer != "Reflection on the _sprint_" {
		t.Fatalf("unexpected markdown answer: %q", decoded.TextAnswer)
	}
}

func TestCanvasMergeSubmissionAnswers_assignmentSubmissionHistory(t *testing.T) {
	// Canvas only exposes other learners' quiz answers to graders through the assignment
	// submission's submission_history[].submission_data. The quiz-submission detail/list blobs
	// come back empty for a grader token, so the assignment submission must populate answers.
	detail := map[string]any{"workflow_state": "complete"}
	assignmentSubmission := map[string]any{
		"user_id": float64(7),
		"submission_history": []any{
			map[string]any{
				"submission_data": []any{
					map[string]any{
						"question_id": float64(101),
						"text":        "10",
						"correct":     false,
						"points":      float64(0),
					},
				},
			},
		},
	}
	merged := canvasMergeSubmissionAnswers([]map[string]any{detail, assignmentSubmission}, nil, nil)
	ans, ok := merged[101]
	if !ok {
		t.Fatalf("expected answer keyed by question 101, got %+v", merged)
	}
	if canvasAnswerAsString(ans.Answer) != "10" {
		t.Fatalf("unexpected answer value: %+v", ans.Answer)
	}
	if ans.Correct == nil || *ans.Correct {
		t.Fatalf("expected correct=false from grader submission_data, got %+v", ans.Correct)
	}
	if ans.Points == nil || *ans.Points != 0 {
		t.Fatalf("expected 0 points, got %+v", ans.Points)
	}
}

func TestCanvasResponseJSONForAnswer_fromMergedHashSubmission(t *testing.T) {
	q := coursemodulequiz.QuizQuestion{
		ID:           "canvas-42",
		QuestionType: "essay",
		Points:       5,
	}
	raw := map[string]any{
		"question_42": "<p>Reflection on the sprint</p>",
	}
	answers := canvasParseSubmissionData(raw)
	if len(answers) != 1 {
		t.Fatalf("unexpected answers: %+v", answers)
	}
	responseJSON := canvasResponseJSONForAnswer(q, answers[0].Answer, nil)
	var payload map[string]any
	if err := json.Unmarshal(responseJSON, &payload); err != nil {
		t.Fatal(err)
	}
	if canvasAnswerAsString(payload["textAnswer"]) == "" {
		t.Fatalf("expected textAnswer in response json, got %+v", payload)
	}
}

func TestCanvasGradeImportedShortAnswerPendingReview(t *testing.T) {
	q := coursemodulequiz.QuizQuestion{
		ID:           "canvas-9",
		QuestionType:   "short_answer",
		Points:         3,
	}
	wrong := false
	answer := canvasQuizSubmissionAnswer{
		CanvasQuestionID: 9,
		Answer:           "student text",
		Correct:          &wrong,
	}
	_, isCorrect, pts, max := canvasGradeImportedQuestion(q, answer, nil, nil)
	if isCorrect != nil {
		t.Fatalf("expected nil is_correct before manual grading, got %v", isCorrect)
	}
	if pts != 0 || max != 3 {
		t.Fatalf("points earned=%v max=%v", pts, max)
	}
}

func TestCanvasGradeImportedNumericWithoutAnswerKeyIsUngraded(t *testing.T) {
	// A reflection/survey numeric question with no defined correct answer (Canvas range 0–0
	// imports as an empty type config). Canvas reports it "incorrect", but it must surface as
	// ungraded, not wrong.
	q := coursemodulequiz.QuizQuestion{
		ID:           "canvas-1",
		QuestionType: "numeric",
		Points:       12,
		TypeConfig:   json.RawMessage(`{}`),
	}
	wrong := false
	zero := 0.0
	answer := canvasQuizSubmissionAnswer{
		CanvasQuestionID: 1,
		Answer:           "10",
		Correct:          &wrong,
		Points:           &zero,
	}
	_, isCorrect, pts, max := canvasGradeImportedQuestion(q, answer, nil, nil)
	if isCorrect != nil {
		t.Fatalf("expected nil is_correct for keyless numeric question, got %v", *isCorrect)
	}
	if pts != 0 || max != 12 {
		t.Fatalf("points earned=%v max=%v", pts, max)
	}
}

func TestCanvasGradeImportedNumericWithAnswerKeyMarksWrong(t *testing.T) {
	q := coursemodulequiz.QuizQuestion{
		ID:           "canvas-2",
		QuestionType: "numeric",
		Points:       4,
		TypeConfig:   json.RawMessage(`{"correct":5,"toleranceAbs":0}`),
	}
	answer := canvasQuizSubmissionAnswer{CanvasQuestionID: 2, Answer: float64(10)}
	_, isCorrect, pts, _ := canvasGradeImportedQuestion(q, answer, nil, nil)
	if isCorrect == nil || *isCorrect {
		t.Fatalf("expected incorrect for keyed numeric question, got %v", isCorrect)
	}
	if pts != 0 {
		t.Fatalf("expected 0 points, got %v", pts)
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
