package httpserver

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

func TestCanvasGradeFromSubmissionPayload_pointsOnly(t *testing.T) {
	sub := map[string]any{
		"user_id": float64(42),
		"score":   8.5,
		"submission_comments": []any{
			map[string]any{
				"author_id":  float64(1),
				"comment":    "Nice work.",
				"created_at": "2024-01-02T00:00:00Z",
			},
			map[string]any{
				"author_id":  float64(42),
				"comment":    "Student reply",
				"created_at": "2024-01-03T00:00:00Z",
			},
		},
	}
	got, err := canvasGradeFromSubmissionPayload(sub, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.hasNumericScore {
		t.Fatal("expected hasNumericScore")
	}
	if got.points != 8.5 {
		t.Fatalf("points=%v want 8.5", got.points)
	}
	if got.comment == nil || *got.comment != "User 1: Nice work.\n\nStudent: Student reply" {
		t.Fatalf("comment=%q", *got.comment)
	}
}

func TestCanvasMapRubricAssessmentScores_matchesByTitle(t *testing.T) {
	critID := uuid.New()
	rubric := &assignmentrubric.RubricDefinition{
		Criteria: []assignmentrubric.RubricCriterion{{
			ID:    critID,
			Title: "Clarity",
			Levels: []assignmentrubric.RubricLevel{
				{Label: "Excellent", Points: 5},
				{Label: "Poor", Points: 0},
			},
		}},
	}
	sub := map[string]any{
		"rubric": []any{
			map[string]any{"id": "_10", "description": "Clarity"},
		},
		"rubric_assessment": map[string]any{
			"score": 4.0,
			"data": map[string]any{
				"_10": map[string]any{"points": 4.0},
			},
		},
	}
	scores, total, ok := canvasMapRubricAssessmentScores(sub, rubric)
	if !ok {
		t.Fatal("expected rubric mapping")
	}
	if scores[critID.String()] != 4.0 {
		t.Fatalf("scores=%v", scores)
	}
	if total != 4.0 {
		t.Fatalf("total=%v", total)
	}
}

func TestCanvasGradeFromSubmissionPayload_enteredScoreUnposted(t *testing.T) {
	sub := map[string]any{
		"user_id":        float64(42),
		"score":          nil,
		"entered_score":  9.0,
		"workflow_state": "graded",
	}
	got, err := canvasGradeFromSubmissionPayload(sub, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.hasNumericScore {
		t.Fatal("expected hasNumericScore from entered_score")
	}
	if got.points != 9.0 {
		t.Fatalf("points=%v want 9", got.points)
	}
}

func TestCanvasMapRubricAssessmentScores_assignmentNestedRubric(t *testing.T) {
	critID := uuid.New()
	rubric := &assignmentrubric.RubricDefinition{
		Criteria: []assignmentrubric.RubricCriterion{{
			ID:    critID,
			Title: "Organization",
			Levels: []assignmentrubric.RubricLevel{{Label: "Good", Points: 10}},
		}},
	}
	sub := map[string]any{
		"assignment": map[string]any{
			"rubric": []any{
				map[string]any{"id": "_42", "description": "Organization"},
			},
		},
		"rubric_assessment": map[string]any{
			"score": 7.0,
			"data": map[string]any{
				"_42": map[string]any{"points": 7.0, "rating_id": "abc"},
			},
		},
	}
	scores, total, ok := canvasMapRubricAssessmentScores(sub, rubric)
	if !ok {
		t.Fatal("expected rubric mapping from nested assignment rubric")
	}
	if scores[critID.String()] != 7.0 {
		t.Fatalf("scores=%v", scores)
	}
	if total != 7.0 {
		t.Fatalf("total=%v", total)
	}
}

func TestCanvasMapRubricAssessmentScores_totalOnlyWhenCriteriaUnmapped(t *testing.T) {
	rubric := &assignmentrubric.RubricDefinition{
		Criteria: []assignmentrubric.RubricCriterion{{
			ID:    uuid.New(),
			Title: "Different title",
			Levels: []assignmentrubric.RubricLevel{{Label: "OK", Points: 5}},
		}},
	}
	sub := map[string]any{
		"rubric_assessment": map[string]any{
			"score": 6.5,
			"data": map[string]any{
				"_99": map[string]any{"points": 6.5},
			},
		},
	}
	_, total, ok := canvasMapRubricAssessmentScores(sub, rubric)
	if !ok {
		t.Fatal("expected total score fallback")
	}
	if total != 6.5 {
		t.Fatalf("total=%v", total)
	}
}

func TestCanvasGradeFromSubmissionPayload_commentOnlyNoScore(t *testing.T) {
	sub := map[string]any{
		"user_id": float64(42),
		"submission_comments": []any{
			map[string]any{
				"author_id":  float64(1),
				"comment":    "Please revise the intro.",
				"created_at": "2024-01-02T00:00:00Z",
			},
		},
	}
	got, err := canvasGradeFromSubmissionPayload(sub, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.hasNumericScore {
		t.Fatal("did not expect numeric score")
	}
	if got.comment == nil || *got.comment != "User 1: Please revise the intro." {
		t.Fatalf("comment=%q", *got.comment)
	}
}

func TestCanvasBuildCanvasGradePushForm_pointsAndComment(t *testing.T) {
	comment := "Strong analysis."
	grade := lexturesGradeForCanvasPush{
		points:     8.5,
		hasNumeric: true,
		comment:    &comment,
	}
	form := canvasBuildCanvasGradePushForm(grade, nil, nil)
	if form.Get("submission[posted_grade]") != "8.5" {
		t.Fatalf("posted_grade=%q", form.Get("submission[posted_grade]"))
	}
	if form.Get("comment[text_comment]") != comment {
		t.Fatalf("comment=%q", form.Get("comment[text_comment]"))
	}
}

func TestCanvasBuildCanvasGradePushForm_rubricByTitle(t *testing.T) {
	critID := uuid.New()
	rubric := &assignmentrubric.RubricDefinition{
		Criteria: []assignmentrubric.RubricCriterion{{
			ID:    critID,
			Title: "Clarity",
			Levels: []assignmentrubric.RubricLevel{
				{Label: "Excellent", Points: 5},
				{Label: "Poor", Points: 0},
			},
		}},
	}
	grade := lexturesGradeForCanvasPush{
		points:       4,
		hasNumeric:   true,
		rubricScores: map[string]float64{critID.String(): 4},
	}
	canvasAssign := map[string]any{
		"rubric": []any{
			map[string]any{
				"id":          "_10",
				"description": "Clarity",
				"ratings": []any{
					map[string]any{"id": "rat1", "points": 5.0},
					map[string]any{"id": "rat2", "points": 0.0},
				},
			},
		},
	}
	comment := "See rubric notes."
	grade.comment = &comment
	form := canvasBuildCanvasGradePushForm(grade, rubric, canvasAssign)
	if form.Get("rubric_assessment[_10][points]") != "4" {
		t.Fatalf("points=%q", form.Get("rubric_assessment[_10][points]"))
	}
	if form.Get("rubric_assessment[_10][rating_id]") != "rat1" {
		t.Fatalf("rating_id=%q", form.Get("rubric_assessment[_10][rating_id]"))
	}
	if form.Get("submission[posted_grade]") != "" {
		t.Fatalf("expected rubric-only form, got posted_grade=%q", form.Get("submission[posted_grade]"))
	}
	if form.Get("comment[text_comment]") != comment {
		t.Fatalf("comment=%q", form.Get("comment[text_comment]"))
	}
}

func TestCanvasBuildCanvasGradePushForm_excused(t *testing.T) {
	grade := lexturesGradeForCanvasPush{excused: true}
	form := canvasBuildCanvasGradePushForm(grade, nil, nil)
	if form.Get("submission[excuse]") != "true" {
		t.Fatalf("excuse=%q", form.Get("submission[excuse]"))
	}
}

func TestCanvasInstructorCommentFromSubmission_readsHistoryComments(t *testing.T) {
	sub := map[string]any{
		"user_id": float64(9),
		"submission_history": []any{
			map[string]any{
				"submission_comments": []any{
					map[string]any{
						"author_id":  float64(2),
						"comment":    "Feedback from an earlier attempt.",
						"created_at": "2024-01-01T00:00:00Z",
					},
				},
			},
		},
	}
	got := canvasInstructorCommentFromSubmission(sub)
	if got != "User 2: Feedback from an earlier attempt." {
		t.Fatalf("got %q", got)
	}
}

func TestCanvasInstructorCommentFromSubmission_includesAllParticipants(t *testing.T) {
	sub := map[string]any{
		"user_id": float64(9),
		"submission_comments": []any{
			map[string]any{"author_id": float64(9), "comment": "mine", "created_at": "2024-01-01T00:00:00Z"},
			map[string]any{"author_id": float64(2), "author_name": "TA Lee", "comment": "grader one", "created_at": "2024-01-02T00:00:00Z"},
			map[string]any{"author_id": float64(3), "author_name": "Prof Kim", "comment": "grader two", "created_at": "2024-01-03T00:00:00Z"},
		},
	}
	got := canvasInstructorCommentFromSubmission(sub)
	want := "Student: mine\n\nTA Lee: grader one\n\nProf Kim: grader two"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCanvasSubmissionCommentsFromPayload_structured(t *testing.T) {
	localUser := uuid.New()
	sub := map[string]any{
		"user_id": float64(42),
		"submission_comments": []any{
			map[string]any{
				"id":         float64(10),
				"author_id":  float64(1),
				"comment":    "Nice work.",
				"created_at": "2024-01-02T00:00:00Z",
				"author": map[string]any{
					"display_name": "Prof Kim",
					"avatar_url":   "https://canvas.example/avatar.png",
				},
			},
			map[string]any{
				"author_id":  float64(42),
				"comment":    "Student reply",
				"created_at": "2024-01-03T00:00:00Z",
			},
		},
	}
	got := canvasSubmissionCommentsFromPayload(sub, map[int64]uuid.UUID{1: localUser})
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].DisplayName != "Prof Kim" || got[0].Body != "Nice work." || got[0].CreatedAt != "2024-01-02T00:00:00Z" {
		t.Fatalf("first=%+v", got[0])
	}
	if got[0].UserID == nil || *got[0].UserID != localUser.String() {
		t.Fatalf("userId=%v", got[0].UserID)
	}
	if got[0].AvatarURL == nil || *got[0].AvatarURL != "https://canvas.example/avatar.png" {
		t.Fatalf("avatar=%v", got[0].AvatarURL)
	}
	if got[1].DisplayName != "Student" || got[1].Body != "Student reply" {
		t.Fatalf("second=%+v", got[1])
	}
}

func TestCanvasNormalizeCanvasSubmissionCommentText_stripsHTML(t *testing.T) {
	got := canvasNormalizeCanvasSubmissionCommentText("<p>Hello <strong>world</strong></p>")
	if got != "Hello world" {
		t.Fatalf("got %q", got)
	}
}