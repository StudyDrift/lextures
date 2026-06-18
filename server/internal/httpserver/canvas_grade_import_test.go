package httpserver

import "testing"

func TestCanvasSubmissionIsGradedForImport_submittedWithHistoryScore(t *testing.T) {
	sub := map[string]any{
		"workflow_state": "submitted",
		"submission_history": []any{
			map[string]any{"score": 9.0},
		},
	}
	if canvasSubmissionIsGradedForImport(sub) {
		t.Fatal("resubmitted workflow_state=submitted should not be graded")
	}
}

func TestSubmissionScoreAndExcused_ignoresHistoryWhenSubmitted(t *testing.T) {
	sub := map[string]any{
		"workflow_state": "submitted",
		"submission_history": []any{
			map[string]any{"score": 9.0},
		},
	}
	if _, _, hasScore := submissionScoreAndExcused(sub); hasScore {
		t.Fatal("expected no score from prior attempt history when submission is ungraded")
	}
}

func TestSubmissionScoreAndExcused_usesHistoryWhenGraded(t *testing.T) {
	sub := map[string]any{
		"workflow_state": "graded",
		"submission_history": []any{
			map[string]any{"score": 7.5},
		},
	}
	_, score, hasScore := submissionScoreAndExcused(sub)
	if !hasScore || score != 7.5 {
		t.Fatalf("graded submission should use history score, got hasScore=%v score=%v", hasScore, score)
	}
}

func TestCanvasSubmissionIsGradedForImport_gradedWorkflow(t *testing.T) {
	sub := map[string]any{"workflow_state": "graded", "score": 10.0}
	if !canvasSubmissionIsGradedForImport(sub) {
		t.Fatal("workflow_state=graded should be graded")
	}
}

func TestCanvasQuizSubmissionIsGradedForImport_pendingReview(t *testing.T) {
	sub := map[string]any{"workflow_state": "pending_review", "kept_score": 8.0}
	if canvasQuizSubmissionIsGradedForImport(sub) {
		t.Fatal("pending_review quiz submission should not be treated as fully graded")
	}
}

func TestCanvasQuizSubmissionIsGradedForImport_complete(t *testing.T) {
	sub := map[string]any{"workflow_state": "complete", "kept_score": 8.0}
	if !canvasQuizSubmissionIsGradedForImport(sub) {
		t.Fatal("complete quiz submission should be treated as graded")
	}
}