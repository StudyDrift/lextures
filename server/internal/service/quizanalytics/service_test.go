package quizanalytics

import (
	"testing"

	"github.com/google/uuid"
	repoitemanalysis "github.com/lextures/lextures/server/internal/repos/itemanalysis"
)

func scorePtr(v float64) *float64 { return &v }

func TestBuildReportScoreHistogram(t *testing.T) {
	quizID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	rows := []repoitemanalysis.AttemptResponseRow{
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), ScorePercent: scorePtr(15), QuestionIndex: 0, MaxPoints: 1, PointsAwarded: 1},
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), ScorePercent: scorePtr(55), QuestionIndex: 0, MaxPoints: 1, PointsAwarded: 0},
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), ScorePercent: scorePtr(95), QuestionIndex: 0, MaxPoints: 1, PointsAwarded: 1},
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000004"), ScorePercent: scorePtr(100), QuestionIndex: 0, MaxPoints: 1, PointsAwarded: 1},
	}

	report := BuildReport(quizID, rows)
	if report.NAttempts != 4 {
		t.Fatalf("expected 4 attempts, got %d", report.NAttempts)
	}
	if report.MeanScore == nil || *report.MeanScore != 66.25 {
		t.Fatalf("expected mean 66.25, got %v", report.MeanScore)
	}
	if report.ScoreBuckets[1].Count != 1 || report.ScoreBuckets[5].Count != 1 ||
		report.ScoreBuckets[9].Count != 2 {
		t.Fatalf("unexpected bucket counts: %+v", report.ScoreBuckets)
	}
}

func TestBuildReportQuestionStats(t *testing.T) {
	quizID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	text := "What is 2+2?"
	correct := true
	rows := []repoitemanalysis.AttemptResponseRow{
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000010"), ScorePercent: scorePtr(100), QuestionIndex: 0, PromptText: &text, MaxPoints: 2, PointsAwarded: 2},
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000011"), ScorePercent: scorePtr(50), QuestionIndex: 0, PromptText: &text, MaxPoints: 2, PointsAwarded: 1},
		{AttemptID: uuid.MustParse("00000000-0000-0000-0000-000000000012"), ScorePercent: scorePtr(0), QuestionIndex: 0, PromptText: &text, IsCorrect: &correct, MaxPoints: 1, PointsAwarded: 0},
	}

	report := BuildReport(quizID, rows)
	if len(report.QuestionStats) != 1 {
		t.Fatalf("expected 1 question stat, got %d", len(report.QuestionStats))
	}
	stat := report.QuestionStats[0]
	if stat.NResponses != 3 {
		t.Fatalf("expected 3 responses, got %d", stat.NResponses)
	}
	if stat.PctCorrect != 50 {
		t.Fatalf("expected 50%% correct, got %v", stat.PctCorrect)
	}
}
