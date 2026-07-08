package derivers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func scorePtr(v float32) *float32 { return &v }

func TestComputeLearningApproach_ProductivePersistence(t *testing.T) {
	itemID := uuid.New()
	courseID := uuid.New()
	started := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	attempts := []quizAttemptRow{
		{AttemptID: uuid.New(), CourseID: courseID, StructureItemID: itemID, AttemptNumber: 1, StartedAt: started, ScorePercent: scorePtr(50)},
		{AttemptID: uuid.New(), CourseID: courseID, StructureItemID: itemID, AttemptNumber: 2, StartedAt: started.Add(time.Hour), ScorePercent: scorePtr(68)},
		{AttemptID: uuid.New(), CourseID: courseID, StructureItemID: uuid.New(), AttemptNumber: 1, StartedAt: started, ScorePercent: scorePtr(70)},
		{AttemptID: uuid.New(), CourseID: courseID, StructureItemID: uuid.New(), AttemptNumber: 2, StartedAt: started.Add(time.Hour), ScorePercent: scorePtr(88)},
		{AttemptID: uuid.New(), CourseID: courseID, StructureItemID: uuid.New(), AttemptNumber: 1, StartedAt: started, ScorePercent: scorePtr(75)},
	}
	summary, sufficient := computeLearningApproach(learningApproachComputeInput{
		QuizAttempts: attempts,
	})
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if summary.Persistence.Level != "high" {
		t.Fatalf("level=%q want high", summary.Persistence.Level)
	}
	if !summary.Persistence.Productive {
		t.Fatal("expected productive persistence")
	}
	if summary.Persistence.AvgScoreDeltaOnRetake <= 0 {
		t.Fatalf("avg delta=%v want positive", summary.Persistence.AvgScoreDeltaOnRetake)
	}
}

func TestComputeLearningApproach_EarlyHintReliance(t *testing.T) {
	attemptID := uuid.New()
	started := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	hints := []hintRequestRow{
		{AttemptID: attemptID, QuestionID: "q1", StartedAt: started, RequestedAt: started.Add(5 * time.Second)},
		{AttemptID: attemptID, QuestionID: "q2", StartedAt: started, RequestedAt: started.Add(8 * time.Second)},
		{AttemptID: uuid.New(), QuestionID: "q3", StartedAt: started, RequestedAt: started.Add(12 * time.Second)},
		{AttemptID: uuid.New(), QuestionID: "q4", StartedAt: started, RequestedAt: started.Add(3 * time.Second)},
	}
	attempts := make([]quizAttemptRow, 5)
	for i := range attempts {
		attempts[i] = quizAttemptRow{
			AttemptID: uuid.New(), CourseID: uuid.New(), StructureItemID: uuid.New(),
			AttemptNumber: 1, StartedAt: started, ScorePercent: scorePtr(80),
		}
	}
	summary, sufficient := computeLearningApproach(learningApproachComputeInput{
		QuizAttempts: attempts,
		HintRequests: hints,
	})
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if summary.HelpSeeking.Style != "early-reliance" {
		t.Fatalf("style=%q want early-reliance", summary.HelpSeeking.Style)
	}
	if summary.HelpSeeking.EarlyHintShare < learningApproachEarlyHintShare {
		t.Fatalf("early share=%v", summary.HelpSeeking.EarlyHintShare)
	}
}

func TestComputeLearningApproach_ActiveConsolidation(t *testing.T) {
	attempts := make([]quizAttemptRow, 5)
	for i := range attempts {
		attempts[i] = quizAttemptRow{
			AttemptID: uuid.New(), CourseID: uuid.New(), StructureItemID: uuid.New(),
			AttemptNumber: 1, StartedAt: time.Now().UTC(), ScorePercent: scorePtr(70),
		}
	}
	summary, sufficient := computeLearningApproach(learningApproachComputeInput{
		QuizAttempts:    attempts,
		NotebookActions: 22,
	})
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if summary.Consolidation.Level != "active" {
		t.Fatalf("level=%q want active", summary.Consolidation.Level)
	}
	if summary.Consolidation.NotebookActions != 22 {
		t.Fatalf("actions=%d", summary.Consolidation.NotebookActions)
	}
}

func TestComputeLearningApproach_InsufficientData(t *testing.T) {
	attempts := []quizAttemptRow{
		{AttemptID: uuid.New(), CourseID: uuid.New(), StructureItemID: uuid.New(), AttemptNumber: 1, StartedAt: time.Now().UTC(), ScorePercent: scorePtr(60)},
		{AttemptID: uuid.New(), CourseID: uuid.New(), StructureItemID: uuid.New(), AttemptNumber: 1, StartedAt: time.Now().UTC(), ScorePercent: scorePtr(70)},
	}
	_, sufficient := computeLearningApproach(learningApproachComputeInput{
		QuizAttempts:    attempts,
		NotebookActions: 0,
	})
	if sufficient {
		t.Fatal("expected insufficient_data")
	}
}

func TestComputeLearningApproach_SufficientViaNotebookOnly(t *testing.T) {
	_, sufficient := computeLearningApproach(learningApproachComputeInput{
		QuizAttempts:    nil,
		NotebookActions: 5,
	})
	if !sufficient {
		t.Fatal("expected sufficient via notebook actions")
	}
}

func TestComputeLearningApproach_NoSingleGritScore(t *testing.T) {
	itemID := uuid.New()
	started := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	attempts := make([]quizAttemptRow, 5)
	for i := range attempts {
		attempts[i] = quizAttemptRow{
			AttemptID: uuid.New(), CourseID: uuid.New(), StructureItemID: itemID,
			AttemptNumber: i + 1, StartedAt: started.Add(time.Duration(i) * time.Hour),
			ScorePercent: scorePtr(float32(50 + i*5)),
		}
	}
	summary, sufficient := computeLearningApproach(learningApproachComputeInput{QuizAttempts: attempts})
	if !sufficient {
		t.Fatal("expected sufficient data")
	}
	if summary.Persistence.Level == "" || summary.HelpSeeking.Style == "" || summary.Consolidation.Level == "" {
		t.Fatalf("missing dimensions: %+v", summary)
	}
}

func TestRetakeMetrics_UnproductiveThrashing(t *testing.T) {
	itemID := uuid.New()
	started := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	attempts := []quizAttemptRow{
		{StructureItemID: itemID, AttemptNumber: 1, ScorePercent: scorePtr(70)},
		{StructureItemID: itemID, AttemptNumber: 2, ScorePercent: scorePtr(65)},
		{StructureItemID: itemID, AttemptNumber: 3, ScorePercent: scorePtr(60)},
	}
	for i := range attempts {
		attempts[i].AttemptID = uuid.New()
		attempts[i].CourseID = uuid.New()
		attempts[i].StartedAt = started
	}
	summary, _ := computeLearningApproach(learningApproachComputeInput{
		QuizAttempts: append(attempts, quizAttemptRow{
			AttemptID: uuid.New(), CourseID: uuid.New(), StructureItemID: uuid.New(),
			AttemptNumber: 1, StartedAt: started, ScorePercent: scorePtr(80),
		}, quizAttemptRow{
			AttemptID: uuid.New(), CourseID: uuid.New(), StructureItemID: uuid.New(),
			AttemptNumber: 1, StartedAt: started, ScorePercent: scorePtr(75),
		}),
	})
	if summary.Persistence.Productive {
		t.Fatal("expected unproductive persistence")
	}
}

func TestHelpSeekingStyle_Independent(t *testing.T) {
	if got := helpSeekingStyle(0.1, 0.1); got != "independent" {
		t.Fatalf("style=%q want independent", got)
	}
}

func TestCountNotebookActions(t *testing.T) {
	pages := []notebookPage{
		{ID: "g1", Kind: "group", Title: "Ecology"},
		{ID: "p1", Kind: "page", ContentMd: "notes"},
		{ID: "p2", Kind: "page", ContentMd: "   "},
	}
	if got := countNotebookActions(pages); got != 1 {
		t.Fatalf("count=%d want 1", got)
	}
}