package derivers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/learnerprofile"
)

// LearningApproachDeriver derives persistence, help-seeking, and consolidation dimensions (LP06).
type LearningApproachDeriver struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

func (d LearningApproachDeriver) Key() string { return "learning_approach" }

func (d LearningApproachDeriver) Version() int { return learningApproachDeriverVersion }

func (d LearningApproachDeriver) MinSignals() int {
	return learningApproachMinQuizAttempts
}

func (d LearningApproachDeriver) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d LearningApproachDeriver) Derive(ctx context.Context, userID uuid.UUID) (learnerprofile.FacetResult, error) {
	now := d.now()
	windowEnd := now
	windowStart := now.AddDate(0, 0, -90)

	quizAttempts, err := d.loadQuizAttempts(ctx, userID)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	hintRequests, err := d.loadHintRequests(ctx, userID)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	notebookActions, err := d.countNotebookActionsForUser(ctx, userID)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}
	revisions, err := d.loadAssignmentRevisions(ctx, userID)
	if err != nil {
		return learnerprofile.FacetResult{}, err
	}

	summary, sufficient := computeLearningApproach(learningApproachComputeInput{
		QuizAttempts:    quizAttempts,
		HintRequests:    hintRequests,
		NotebookActions: notebookActions,
		AssignmentRevs:  revisions,
	})
	if !sufficient {
		return learnerprofile.FacetResult{
			State:           "insufficient_data",
			Summary:         json.RawMessage(`{}`),
			Confidence:      0,
			ComputedVersion: d.Version(),
		}, nil
	}

	summaryJSON, _ := json.Marshal(summary)
	confidence := learningApproachConfidence(summary, len(quizAttempts), len(hintRequests), notebookActions)
	insights := buildLearningApproachInsights(summary, quizAttempts, hintRequests, notebookActions, revisions, windowStart, windowEnd)

	return learnerprofile.FacetResult{
		State:           "ok",
		Summary:         summaryJSON,
		Confidence:      confidence,
		ComputedVersion: d.Version(),
		Insights:        insights,
	}, nil
}

func (d LearningApproachDeriver) loadQuizAttempts(ctx context.Context, userID uuid.UUID) ([]quizAttemptRow, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT
    qa.id,
    qa.course_id,
    qa.structure_item_id,
    qa.attempt_number,
    qa.started_at,
    qa.score_percent
FROM course.quiz_attempts qa
INNER JOIN course.courses co ON co.id = qa.course_id
INNER JOIN course.course_enrollments ce ON ce.course_id = qa.course_id AND ce.user_id = qa.student_user_id
WHERE qa.student_user_id = $1
  AND qa.status = 'submitted'
  AND co.archived = false
  AND (ce.active OR ce.state = 'active')
ORDER BY qa.structure_item_id, qa.attempt_number
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []quizAttemptRow
	for rows.Next() {
		var row quizAttemptRow
		if err := rows.Scan(
			&row.AttemptID, &row.CourseID, &row.StructureItemID,
			&row.AttemptNumber, &row.StartedAt, &row.ScorePercent,
		); err != nil {
			return nil, err
		}
		row.StartedAt = row.StartedAt.UTC()
		out = append(out, row)
	}
	return out, rows.Err()
}

func (d LearningApproachDeriver) loadHintRequests(ctx context.Context, userID uuid.UUID) ([]hintRequestRow, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT
    hr.attempt_id,
    hr.question_id,
    hr.requested_at,
    qa.started_at
FROM course.hint_requests hr
INNER JOIN course.quiz_attempts qa ON qa.id = hr.attempt_id
INNER JOIN course.course_enrollments ce ON ce.course_id = qa.course_id AND ce.user_id = qa.student_user_id
WHERE qa.student_user_id = $1
  AND qa.status = 'submitted'
  AND (ce.active OR ce.state = 'active')
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []hintRequestRow
	for rows.Next() {
		var row hintRequestRow
		if err := rows.Scan(&row.AttemptID, &row.QuestionID, &row.RequestedAt, &row.StartedAt); err != nil {
			return nil, err
		}
		row.RequestedAt = row.RequestedAt.UTC()
		row.StartedAt = row.StartedAt.UTC()
		out = append(out, row)
	}
	return out, rows.Err()
}

func (d LearningApproachDeriver) countNotebookActionsForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT data FROM analytics.student_notebooks WHERE user_id = $1
`, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return 0, err
		}
		var store notebookStore
		if err := json.Unmarshal(data, &store); err != nil {
			continue
		}
		total += countNotebookActions(store.Pages)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var taskCount int
	if err := d.Pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM analytics.student_notebook_tasks WHERE user_id = $1
`, userID).Scan(&taskCount); err != nil {
		return 0, err
	}
	return total + taskCount, nil
}

func (d LearningApproachDeriver) loadAssignmentRevisions(ctx context.Context, userID uuid.UUID) ([]revisionRow, error) {
	rows, err := d.Pool.Query(ctx, `
SELECT sv.course_id, sv.module_item_id, sv.version_number, sv.submitted_at
FROM course.submission_versions sv
INNER JOIN course.course_enrollments ce ON ce.course_id = sv.course_id AND ce.user_id = sv.student_id
WHERE sv.student_id = $1
  AND (ce.active OR ce.state = 'active')
UNION ALL
SELECT mas.course_id, mas.module_item_id, mas.version_number, mas.submitted_at
FROM course.module_assignment_submissions mas
INNER JOIN course.course_enrollments ce ON ce.course_id = mas.course_id AND ce.user_id = mas.submitted_by
WHERE mas.submitted_by = $1
  AND (ce.active OR ce.state = 'active')
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []revisionRow
	for rows.Next() {
		var row revisionRow
		if err := rows.Scan(&row.CourseID, &row.ModuleItemID, &row.VersionNumber, &row.SubmittedAt); err != nil {
			return nil, err
		}
		row.SubmittedAt = row.SubmittedAt.UTC()
		out = append(out, row)
	}
	return out, rows.Err()
}

func buildLearningApproachInsights(
	summary LearningApproachSummary,
	quizAttempts []quizAttemptRow,
	hintRequests []hintRequestRow,
	notebookActions int,
	revisions []revisionRow,
	windowStart, windowEnd time.Time,
) []learnerprofile.InsightResult {
	ws := windowStart
	we := windowEnd
	insights := []learnerprofile.InsightResult{
		buildPersistenceInsight(summary.Persistence, quizAttempts, revisions, ws, we),
		buildHelpSeekingInsight(summary.HelpSeeking, hintRequests, ws, we),
		buildConsolidationInsight(summary.Consolidation, notebookActions, ws, we),
	}
	return insights
}

func buildPersistenceInsight(
	dim PersistenceDimension,
	attempts []quizAttemptRow,
	revisions []revisionRow,
	windowStart, windowEnd time.Time,
) learnerprofile.InsightResult {
	value, _ := json.Marshal(dim)
	_, _, deltas := retakeMetrics(attempts)
	evidence := []learnerprofile.EvidenceResult{{
		SourceKind:       "quiz_attempt",
		SourceTable:      learningApproachQuizSourceTable,
		ObservationCount: len(attempts),
		WindowStart:      &windowStart,
		WindowEnd:        &windowEnd,
	}}
	if len(deltas) > 0 {
		sample, _ := json.Marshal(map[string]any{
			"scoreDeltas": deltas,
			"retakeRate":  dim.RetakeRate,
		})
		evidence[0].SampleRefs = sample
	}
	if len(revisions) > 0 {
		contrib := round2(dim.RevisionRate)
		evidence = append(evidence, learnerprofile.EvidenceResult{
			SourceKind:       "assignment_revision",
			SourceTable:      learningApproachRevisionTable,
			ObservationCount: len(revisions),
			Contribution:     &contrib,
			WindowStart:        &windowStart,
			WindowEnd:          &windowEnd,
		})
	}
	byCourse := make(map[uuid.UUID]int)
	for _, row := range attempts {
		byCourse[row.CourseID]++
	}
	for courseID, count := range byCourse {
		contrib := 1.0
		if len(attempts) > 0 {
			contrib = round2(float64(count) / float64(len(attempts)))
		}
		ev := learnerprofile.EvidenceResult{
			SourceKind:       "quiz_attempt",
			SourceTable:      learningApproachQuizSourceTable,
			CourseID:         &courseID,
			ObservationCount: count,
			Contribution:     &contrib,
			WindowStart:      &windowStart,
			WindowEnd:        &windowEnd,
		}
		evidence = append(evidence, ev)
	}
	confidence := dim.RetakeRate
	if dim.Productive {
		confidence = mathMin(1, dim.RetakeRate+dim.AvgScoreDeltaOnRetake)
	}
	return learnerprofile.InsightResult{
		InsightKey:   "persistence",
		LabelI18nKey: "learner_profile.learning_approach.persistence",
		Value:        value,
		Confidence:   round2(confidence),
		Salience:     100,
		Evidence:     evidence,
	}
}

func buildHelpSeekingInsight(
	dim HelpSeekingDimension,
	hints []hintRequestRow,
	windowStart, windowEnd time.Time,
) learnerprofile.InsightResult {
	value, _ := json.Marshal(dim)
	evidence := []learnerprofile.EvidenceResult{{
		SourceKind:       "hint_request",
		SourceTable:      learningApproachHintSourceTable,
		ObservationCount: len(hints),
		WindowStart:      &windowStart,
		WindowEnd:        &windowEnd,
	}}
	if len(hints) > 0 {
		early := 0
		timingSamples := make([]map[string]any, 0, minInt(8, len(hints)))
		for _, hint := range hints {
			elapsedSec := int(hint.RequestedAt.Sub(hint.StartedAt).Seconds())
			if elapsedSec <= learningApproachEarlyHintSec {
				early++
			}
			if len(timingSamples) < 8 {
				timingSamples = append(timingSamples, map[string]any{
					"questionId":  hint.QuestionID,
					"elapsedSec":  elapsedSec,
					"attemptId":   hint.AttemptID.String(),
				})
			}
		}
		sample, _ := json.Marshal(map[string]any{
			"earlyHintCount": early,
			"hintCount":      len(hints),
			"timingSamples":  timingSamples,
		})
		evidence[0].SampleRefs = sample
	}
	confidence := 0.5
	if len(hints) > 0 {
		confidence = dim.EarlyHintShare
		if dim.Style == "independent" {
			confidence = 0.75
		}
	}
	return learnerprofile.InsightResult{
		InsightKey:   "help_seeking",
		LabelI18nKey: "learner_profile.learning_approach.help_seeking",
		Value:        value,
		Confidence:   round2(confidence),
		Salience:     90,
		Evidence:     evidence,
	}
}

func buildConsolidationInsight(
	dim ConsolidationDimension,
	notebookActions int,
	windowStart, windowEnd time.Time,
) learnerprofile.InsightResult {
	value, _ := json.Marshal(dim)
	contrib := 1.0
	evidence := []learnerprofile.EvidenceResult{
		{
			SourceKind:       "notebook_page",
			SourceTable:      learningApproachNotebookTable,
			ObservationCount: notebookActions,
			Contribution:     &contrib,
			WindowStart:      &windowStart,
			WindowEnd:        &windowEnd,
		},
		{
			SourceKind:       "notebook_task",
			SourceTable:      learningApproachTaskTable,
			ObservationCount: notebookActions,
			Contribution:     &contrib,
			WindowStart:      &windowStart,
			WindowEnd:        &windowEnd,
		},
	}
	confidence := 0.25
	if notebookActions >= learningApproachModerateNotebook {
		confidence = float64(notebookActions) / float64(learningApproachActiveNotebook)
		if confidence > 1 {
			confidence = 1
		}
	}
	return learnerprofile.InsightResult{
		InsightKey:   "consolidation",
		LabelI18nKey: "learner_profile.learning_approach.consolidation",
		Value:        value,
		Confidence:   round2(confidence),
		Salience:     80,
		Evidence:     evidence,
	}
}

func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}