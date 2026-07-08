package derivers

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	learningApproachDeriverVersion   = 1
	learningApproachMinQuizAttempts  = 5
	learningApproachMinNotebookActs  = 3
	learningApproachEarlyHintSec     = 30
	learningApproachProductiveDelta  = 0.03
	learningApproachHighRetakeRate   = 0.35
	learningApproachLowRetakeRate    = 0.15
	learningApproachActiveNotebook   = 10
	learningApproachModerateNotebook = 3
	learningApproachEarlyHintShare   = 0.6
	learningApproachIndependentHints = 0.3
	learningApproachQuizSourceTable  = "course.quiz_attempts"
	learningApproachHintSourceTable  = "course.hint_requests"
	learningApproachNotebookTable    = "analytics.student_notebooks"
	learningApproachTaskTable        = "analytics.student_notebook_tasks"
	learningApproachRevisionTable    = "course.submission_versions"
)

// PersistenceDimension describes retake and revision behaviour.
type PersistenceDimension struct {
	Level                 string  `json:"level"`
	Productive            bool    `json:"productive"`
	RetakeRate            float64 `json:"retakeRate"`
	AvgScoreDeltaOnRetake float64 `json:"avgScoreDeltaOnRetake"`
	RevisionRate          float64 `json:"revisionRate,omitempty"`
	QuizAttemptCount      int     `json:"quizAttemptCount,omitempty"`
}

// HelpSeekingDimension describes hint timing and frequency.
type HelpSeekingDimension struct {
	Style           string  `json:"style"`
	HintsPerAttempt float64 `json:"hintsPerAttempt"`
	EarlyHintShare  float64 `json:"earlyHintShare,omitempty"`
}

// ConsolidationDimension describes notebook and flashcard-style consolidation activity.
type ConsolidationDimension struct {
	Level           string `json:"level"`
	NotebookActions int    `json:"notebookActions"`
}

// LearningApproachSummary is the facet-level aggregate returned in summary JSON.
type LearningApproachSummary struct {
	Persistence   PersistenceDimension   `json:"persistence"`
	HelpSeeking   HelpSeekingDimension   `json:"helpSeeking"`
	Consolidation ConsolidationDimension `json:"consolidation"`
}

type quizAttemptRow struct {
	AttemptID       uuid.UUID
	CourseID        uuid.UUID
	StructureItemID uuid.UUID
	AttemptNumber   int
	StartedAt       time.Time
	ScorePercent    *float32
}

type hintRequestRow struct {
	AttemptID   uuid.UUID
	QuestionID  string
	RequestedAt time.Time
	StartedAt   time.Time
}

type revisionRow struct {
	CourseID      uuid.UUID
	ModuleItemID  uuid.UUID
	VersionNumber int
	SubmittedAt   time.Time
}

type learningApproachComputeInput struct {
	QuizAttempts     []quizAttemptRow
	HintRequests     []hintRequestRow
	NotebookActions  int
	AssignmentRevs   []revisionRow
}

func computeLearningApproach(in learningApproachComputeInput) (LearningApproachSummary, bool) {
	quizCount := len(in.QuizAttempts)
	if quizCount < learningApproachMinQuizAttempts && in.NotebookActions < learningApproachMinNotebookActs {
		return LearningApproachSummary{}, false
	}

	persistence := computePersistence(in.QuizAttempts, in.AssignmentRevs)
	helpSeeking := computeHelpSeeking(in.HintRequests, len(in.QuizAttempts))
	consolidation := computeConsolidation(in.NotebookActions)

	return LearningApproachSummary{
		Persistence:   persistence,
		HelpSeeking:   helpSeeking,
		Consolidation: consolidation,
	}, true
}

func computePersistence(attempts []quizAttemptRow, revisions []revisionRow) PersistenceDimension {
	retakeRate, avgDelta, _ := retakeMetrics(attempts)
	revisionRate := assignmentRevisionRate(revisions)

	productive := avgDelta > learningApproachProductiveDelta
	level := persistenceLevel(retakeRate, avgDelta, productive)

	return PersistenceDimension{
		Level:                 level,
		Productive:            productive,
		RetakeRate:            round2(retakeRate),
		AvgScoreDeltaOnRetake: round2(avgDelta),
		RevisionRate:          round2(revisionRate),
		QuizAttemptCount:      len(attempts),
	}
}

func retakeMetrics(attempts []quizAttemptRow) (retakeRate float64, avgDelta float64, deltas []float64) {
	if len(attempts) == 0 {
		return 0, 0, nil
	}

	type itemKey string
	byItem := make(map[itemKey][]quizAttemptRow)
	for _, row := range attempts {
		key := itemKey(row.StructureItemID.String())
		byItem[key] = append(byItem[key], row)
	}

	retakenItems := 0
	for key, rows := range byItem {
		sort.Slice(rows, func(i, j int) bool { return rows[i].AttemptNumber < rows[j].AttemptNumber })
		byItem[key] = rows
		if len(rows) > 1 || (len(rows) == 1 && rows[0].AttemptNumber > 1) {
			retakenItems++
		}
		for i := 1; i < len(rows); i++ {
			prev, cur := rows[i-1].ScorePercent, rows[i].ScorePercent
			if prev == nil || cur == nil {
				continue
			}
			delta := float64(*cur) - float64(*prev)
			deltas = append(deltas, delta)
		}
	}

	itemCount := len(byItem)
	if itemCount > 0 {
		retakeRate = float64(retakenItems) / float64(itemCount)
	}
	if len(deltas) > 0 {
		sum := 0.0
		for _, d := range deltas {
			sum += d
		}
		avgDelta = sum / float64(len(deltas))
	}
	return retakeRate, avgDelta, deltas
}

func persistenceLevel(retakeRate, avgDelta float64, productive bool) string {
	if retakeRate >= learningApproachHighRetakeRate && productive {
		return "high"
	}
	if retakeRate < learningApproachLowRetakeRate {
		return "low"
	}
	if retakeRate >= learningApproachLowRetakeRate && avgDelta <= 0 && !productive {
		return "low"
	}
	if retakeRate >= learningApproachHighRetakeRate {
		return "medium"
	}
	return "medium"
}

func assignmentRevisionRate(revisions []revisionRow) float64 {
	if len(revisions) == 0 {
		return 0
	}
	maxVersion := make(map[string]int)
	for _, row := range revisions {
		key := row.ModuleItemID.String()
		if row.VersionNumber > maxVersion[key] {
			maxVersion[key] = row.VersionNumber
		}
	}
	if len(maxVersion) == 0 {
		return 0
	}
	revised := 0
	for _, ver := range maxVersion {
		if ver > 1 {
			revised++
		}
	}
	return float64(revised) / float64(len(maxVersion))
}

func computeHelpSeeking(hints []hintRequestRow, attemptCount int) HelpSeekingDimension {
	if len(hints) == 0 {
		return HelpSeekingDimension{
			Style:           "independent",
			HintsPerAttempt: 0,
		}
	}

	attemptHints := make(map[uuid.UUID]int)
	earlyHints := 0
	for _, hint := range hints {
		attemptHints[hint.AttemptID]++
		elapsed := hint.RequestedAt.Sub(hint.StartedAt)
		if elapsed <= learningApproachEarlyHintSec*time.Second {
			earlyHints++
		}
	}

	hintsPerAttempt := float64(len(hints)) / float64(len(attemptHints))
	earlyShare := float64(earlyHints) / float64(len(hints))
	style := helpSeekingStyle(hintsPerAttempt, earlyShare)

	return HelpSeekingDimension{
		Style:           style,
		HintsPerAttempt: round2(hintsPerAttempt),
		EarlyHintShare:  round2(earlyShare),
	}
}

func helpSeekingStyle(hintsPerAttempt, earlyShare float64) string {
	if earlyShare >= learningApproachEarlyHintShare {
		return "early-reliance"
	}
	if hintsPerAttempt < learningApproachIndependentHints && earlyShare < learningApproachEarlyHintShare/2 {
		return "independent"
	}
	return "balanced"
}

func computeConsolidation(notebookActions int) ConsolidationDimension {
	level := "light"
	switch {
	case notebookActions >= learningApproachActiveNotebook:
		level = "active"
	case notebookActions >= learningApproachModerateNotebook:
		level = "moderate"
	}
	return ConsolidationDimension{
		Level:           level,
		NotebookActions: notebookActions,
	}
}

func learningApproachConfidence(summary LearningApproachSummary, quizCount, hintCount, notebookActions int) float64 {
	if quizCount < learningApproachMinQuizAttempts && notebookActions < learningApproachMinNotebookActs {
		return 0
	}
	quizFactor := math.Min(1, float64(quizCount)/15.0)
	notebookFactor := math.Min(1, float64(notebookActions)/15.0)
	hintFactor := 0.5
	if hintCount > 0 {
		hintFactor = math.Min(1, float64(hintCount)/10.0)
	}
	signalFactor := math.Max(quizFactor, notebookFactor)
	if hintCount > 0 {
		signalFactor = (signalFactor + hintFactor) / 2
	}
	dimensionCount := 0
	if summary.Persistence.QuizAttemptCount > 0 {
		dimensionCount++
	}
	if summary.HelpSeeking.HintsPerAttempt > 0 || hintCount == 0 {
		dimensionCount++
	}
	if summary.Consolidation.NotebookActions > 0 || notebookActions == 0 {
		dimensionCount++
	}
	dimFactor := math.Min(1, float64(dimensionCount)/3.0)
	return round2(math.Max(0.25, signalFactor*dimFactor))
}

func countNotebookActions(pages []notebookPage) int {
	n := 0
	for _, page := range pages {
		if page.Kind == "group" {
			continue
		}
		if strings.TrimSpace(page.ContentMd) != "" {
			n++
		}
	}
	return n
}