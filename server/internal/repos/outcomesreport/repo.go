// Package outcomesreport aggregates cohort achievement on course learning outcomes (plan 9.5).
package outcomesreport

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
)

const refreshStaleAfter = time.Hour

// RefreshMeta holds when the outcomes cache was last refreshed globally.
type RefreshMeta struct {
	RefreshedAt time.Time
}

// StudentEnrollment is an active student row used during refresh.
type StudentEnrollment struct {
	UserID       uuid.UUID
	EnrollmentID uuid.UUID
	SectionID    *uuid.UUID
}

// OutcomeRow is one outcome in a cohort report.
type OutcomeRow struct {
	OutcomeID       uuid.UUID
	Title           string
	SortOrder       int32
	NStudents       int
	NAssessed       int
	MeanScore       *float32
	PctMet          float64
	PctNotMet       float64
	Threshold       float32
	AlignmentCount  int
	ImprovementNote string
	NoAlignments    bool
}

// ReportResult is the full outcomes analytics payload.
type ReportResult struct {
	CourseID          uuid.UUID
	MasteryThreshold  float32
	DataAsOf          time.Time
	StaleMinutes      int
	Outcomes          []OutcomeRow
}

// GetRefreshMeta returns global refresh timing.
func GetRefreshMeta(ctx context.Context, pool *pgxpool.Pool) (RefreshMeta, error) {
	var t time.Time
	err := pool.QueryRow(ctx, `SELECT refreshed_at FROM analytics.outcomes_report_refresh WHERE id = 1`).Scan(&t)
	if err != nil {
		return RefreshMeta{}, err
	}
	return RefreshMeta{RefreshedAt: t.UTC()}, nil
}

// RefreshViewIfStale rebuilds caches when older than refreshStaleAfter.
func RefreshViewIfStale(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	meta, err := GetRefreshMeta(ctx, pool)
	if err != nil {
		return false, err
	}
	if time.Since(meta.RefreshedAt) < refreshStaleAfter {
		return false, nil
	}
	return true, RefreshAllCourses(ctx, pool)
}

// RefreshCourseNow rebuilds student rows and the materialized view for one course.
func RefreshCourseNow(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	if err := rebuildStudentRows(ctx, pool, courseID); err != nil {
		return err
	}
	return refreshMaterializedView(ctx, pool)
}

// RefreshAllCourses rebuilds all courses (used by scheduled refresh).
func RefreshAllCourses(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `SELECT id FROM course.courses`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if err := rebuildStudentRows(ctx, pool, id); err != nil {
			return err
		}
	}
	return refreshMaterializedView(ctx, pool)
}

func refreshMaterializedView(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.outcomes_report`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE analytics.outcomes_report_refresh SET refreshed_at = NOW() WHERE id = 1`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetMasteryThreshold returns the configured threshold for a course (default 70).
func GetMasteryThreshold(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (float32, error) {
	var t float32
	err := pool.QueryRow(ctx, `
SELECT COALESCE(
    (SELECT mastery_threshold FROM analytics.course_outcomes_report_config WHERE course_id = $1),
    70.0
)::real
`, courseID).Scan(&t)
	return t, err
}

// SetMasteryThreshold upserts the course threshold.
func SetMasteryThreshold(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, threshold float32) error {
	if threshold <= 0 || threshold > 100 {
		return fmt.Errorf("outcomesreport: threshold out of range")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.course_outcomes_report_config (course_id, mastery_threshold, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (course_id) DO UPDATE SET mastery_threshold = EXCLUDED.mastery_threshold, updated_at = NOW()
`, courseID, threshold)
	return err
}

// UpsertImprovementNote saves narrative text for an outcome.
func UpsertImprovementNote(ctx context.Context, pool *pgxpool.Pool, outcomeID uuid.UUID, text string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.outcome_improvement_notes (outcome_id, note_text, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (outcome_id) DO UPDATE SET note_text = EXCLUDED.note_text, updated_at = NOW()
`, outcomeID, text)
	return err
}

// GetImprovementNote returns note text or empty string.
func GetImprovementNote(ctx context.Context, pool *pgxpool.Pool, outcomeID uuid.UUID) (string, error) {
	var note string
	err := pool.QueryRow(ctx, `
SELECT COALESCE(note_text, '') FROM analytics.outcome_improvement_notes WHERE outcome_id = $1
`, outcomeID).Scan(&note)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return note, err
}

// ListActiveStudents returns enrolled students optionally filtered by section or group.
func ListActiveStudents(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	sectionID *uuid.UUID,
	groupID *uuid.UUID,
) ([]StudentEnrollment, error) {
	args := []any{courseID}
	sectionClause := ""
	groupJoin := ""
	if sectionID != nil {
		args = append(args, *sectionID)
		sectionClause = fmt.Sprintf(" AND e.section_id = $%d", len(args))
	}
	if groupID != nil {
		args = append(args, *groupID)
		groupJoin = fmt.Sprintf(`
INNER JOIN course.enrollment_group_memberships egm ON egm.enrollment_id = e.id AND egm.group_id = $%d
`, len(args))
	}
	q := fmt.Sprintf(`
SELECT e.user_id, e.id, e.section_id
FROM course.course_enrollments e
%s
WHERE e.course_id = $1 AND e.active = TRUE AND e.role = 'student'
%s
ORDER BY e.created_at ASC
`, groupJoin, sectionClause)
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudentEnrollment
	for rows.Next() {
		var r StudentEnrollment
		if err := rows.Scan(&r.UserID, &r.EnrollmentID, &r.SectionID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReportForCourse builds a filtered cohort report from cached student rows.
func ReportForCourse(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	sectionID *uuid.UUID,
	groupID *uuid.UUID,
) (ReportResult, error) {
	threshold, err := GetMasteryThreshold(ctx, pool, courseID)
	if err != nil {
		return ReportResult{}, err
	}
	meta, err := GetRefreshMeta(ctx, pool)
	if err != nil {
		return ReportResult{}, err
	}

	outcomes, err := courseoutcomes.ListOutcomes(ctx, pool, courseID)
	if err != nil {
		return ReportResult{}, err
	}
	allLinks, err := courseoutcomes.ListLinksForCourse(ctx, pool, courseID)
	if err != nil {
		return ReportResult{}, err
	}
	linksByOutcome := map[uuid.UUID][]courseoutcomes.OutcomeLinkWithItemRow{}
	for i := range allLinks {
		oid := allLinks[i].OutcomeID
		linksByOutcome[oid] = append(linksByOutcome[oid], allLinks[i])
	}

	students, err := ListActiveStudents(ctx, pool, courseID, sectionID, groupID)
	if err != nil {
		return ReportResult{}, err
	}
	nStudents := len(students)

	notes, err := listImprovementNotes(ctx, pool, courseID)
	if err != nil {
		return ReportResult{}, err
	}

	rows := make([]OutcomeRow, 0, len(outcomes))
	for _, lo := range outcomes {
		links := linksByOutcome[lo.ID]
		nAlign := len(links)
		note := notes[lo.ID]
		if nAlign == 0 {
			rows = append(rows, OutcomeRow{
				OutcomeID:       lo.ID,
				Title:           lo.Title,
				SortOrder:       lo.SortOrder,
				NStudents:       nStudents,
				NAssessed:       0,
				Threshold:       threshold,
				AlignmentCount:  0,
				ImprovementNote: note,
				NoAlignments:    true,
			})
			continue
		}

		stats, err := aggregateOutcome(ctx, pool, lo.ID, students, threshold)
		if err != nil {
			return ReportResult{}, err
		}
		pctMet, pctNotMet := pctMetNotMet(stats.nAssessed, stats.nMet)
		rows = append(rows, OutcomeRow{
			OutcomeID:       lo.ID,
			Title:           lo.Title,
			SortOrder:       lo.SortOrder,
			NStudents:       nStudents,
			NAssessed:       stats.nAssessed,
			MeanScore:       stats.meanScore,
			PctMet:          pctMet,
			PctNotMet:       pctNotMet,
			Threshold:       threshold,
			AlignmentCount:  nAlign,
			ImprovementNote: note,
			NoAlignments:    false,
		})
	}

	return ReportResult{
		CourseID:         courseID,
		MasteryThreshold: threshold,
		DataAsOf:         meta.RefreshedAt,
		StaleMinutes:     int(time.Since(meta.RefreshedAt).Minutes()),
		Outcomes:         rows,
	}, nil
}

type aggStats struct {
	nAssessed int
	nMet      int
	meanScore *float32
}

func aggregateOutcome(
	ctx context.Context,
	pool *pgxpool.Pool,
	outcomeID uuid.UUID,
	students []StudentEnrollment,
	threshold float32,
) (aggStats, error) {
	if len(students) == 0 {
		return aggStats{}, nil
	}
	userIDs := make([]uuid.UUID, len(students))
	for i := range students {
		userIDs[i] = students[i].UserID
	}
	rows, err := pool.Query(ctx, `
SELECT user_id, avg_score_pct, assessed, met
FROM analytics.outcomes_report_student
WHERE outcome_id = $1 AND user_id = ANY($2)
`, outcomeID, userIDs)
	if err != nil {
		return aggStats{}, err
	}
	defer rows.Close()
	var assessed int
	var met int
	var sum float64
	for rows.Next() {
		var uid uuid.UUID
		var avg *float32
		var isAssessed, isMet bool
		if err := rows.Scan(&uid, &avg, &isAssessed, &isMet); err != nil {
			return aggStats{}, err
		}
		if !isAssessed {
			continue
		}
		assessed++
		if isMet {
			met++
		}
		if avg != nil && isFiniteF64(float64(*avg)) {
			sum += float64(*avg)
		}
	}
	if err := rows.Err(); err != nil {
		return aggStats{}, err
	}
	var mean *float32
	if assessed > 0 {
		m := float32(sum / float64(assessed))
		mean = &m
	}
	_ = threshold
	return aggStats{nAssessed: assessed, nMet: met, meanScore: mean}, nil
}

func listImprovementNotes(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := pool.Query(ctx, `
SELECT n.outcome_id, n.note_text
FROM analytics.outcome_improvement_notes n
INNER JOIN course.course_learning_outcomes lo ON lo.id = n.outcome_id
WHERE lo.course_id = $1
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}
		out[id] = text
	}
	return out, rows.Err()
}

func pctMetNotMet(nAssessed, nMet int) (pctMet, pctNotMet float64) {
	if nAssessed <= 0 {
		return 0, 0
	}
	pctMet = math.Round((float64(nMet)/float64(nAssessed))*1000) / 10
	pctNotMet = math.Round((float64(nAssessed-nMet)/float64(nAssessed))*1000) / 10
	return pctMet, pctNotMet
}

func rebuildStudentRows(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	threshold, err := GetMasteryThreshold(ctx, pool, courseID)
	if err != nil {
		return err
	}
	outcomes, err := courseoutcomes.ListOutcomes(ctx, pool, courseID)
	if err != nil {
		return err
	}
	allLinks, err := courseoutcomes.ListLinksForCourse(ctx, pool, courseID)
	if err != nil {
		return err
	}
	linksByOutcome := map[uuid.UUID][]courseoutcomes.OutcomeLinkWithItemRow{}
	for i := range allLinks {
		oid := allLinks[i].OutcomeID
		linksByOutcome[oid] = append(linksByOutcome[oid], allLinks[i])
	}
	students, err := ListActiveStudents(ctx, pool, courseID, nil, nil)
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM analytics.outcomes_report_student WHERE course_id = $1`, courseID); err != nil {
		return err
	}

	for _, stu := range students {
		for _, lo := range outcomes {
			links := linksByOutcome[lo.ID]
			if len(links) == 0 {
				continue
			}
			scores, err := studentScoresForLinks(ctx, tx, courseID, stu.UserID, links)
			if err != nil {
				return err
			}
			avg := WeightedAvgForStudentLinks(links, scores)
			assessed := avg != nil
			met := studentMet(avg, threshold)
			_, err = tx.Exec(ctx, `
INSERT INTO analytics.outcomes_report_student (
    course_id, outcome_id, user_id, enrollment_id, section_id,
    avg_score_pct, assessed, met
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
`, courseID, lo.ID, stu.UserID, stu.EnrollmentID, stu.SectionID, avg, assessed, met)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func studentScoresForLinks(
	ctx context.Context,
	q pgx.Tx,
	courseID, userID uuid.UUID,
	links []courseoutcomes.OutcomeLinkWithItemRow,
) (map[evidenceKey]float32, error) {
	out := map[evidenceKey]float32{}
	for _, link := range links {
		k := evidenceKey{
			structureItemID: link.StructureItemID,
			targetKind:      link.TargetKind,
			quizQuestionID:  link.QuizQuestionID,
			subOutcomeID:    link.SubOutcomeID,
		}
		var pct *float32
		var err error
		switch link.TargetKind {
		case "quiz_question":
			pct, err = scorePercentQuizQuestion(ctx, q, courseID, userID, link.StructureItemID, link.QuizQuestionID)
		case "assignment", "quiz":
			pct, err = scorePercentGradedItem(ctx, q, courseID, userID, link.StructureItemID, link.ItemKind)
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
		if pct != nil {
			out[k] = *pct
		}
	}
	return out, nil
}

func scorePercentGradedItem(
	ctx context.Context,
	q pgx.Tx,
	courseID, userID, itemID uuid.UUID,
	itemKind string,
) (*float32, error) {
	var earned *float64
	var worth *int32
	err := q.QueryRow(ctx, `
SELECT cg.points_earned, COALESCE(ma.points_worth, mq.points_worth)
FROM course.course_grades cg
INNER JOIN course.course_structure_items csi ON csi.id = cg.module_item_id
LEFT JOIN course.module_assignments ma ON ma.structure_item_id = csi.id
LEFT JOIN course.module_quizzes mq ON mq.structure_item_id = csi.id
WHERE cg.course_id = $1 AND cg.student_user_id = $2 AND cg.module_item_id = $3
`, courseID, userID, itemID).Scan(&earned, &worth)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if earned == nil || worth == nil || *worth <= 0 {
		return nil, nil
	}
	p := float32((*earned / float64(*worth)) * 100.0)
	if p < 0 {
		p = 0
	} else if p > 100 {
		p = 100
	}
	_ = itemKind
	return &p, nil
}

func scorePercentQuizQuestion(
	ctx context.Context,
	q pgx.Tx,
	courseID, userID, quizItemID uuid.UUID,
	questionID string,
) (*float32, error) {
	var ratio *float64
	err := q.QueryRow(ctx, `
WITH latest AS (
    SELECT id FROM course.quiz_attempts
    WHERE course_id = $1 AND structure_item_id = $2 AND student_user_id = $3 AND status = 'submitted'
    ORDER BY submitted_at DESC NULLS LAST, id DESC
    LIMIT 1
)
SELECT
    CASE WHEN qr.max_points > 0::double precision
        THEN (COALESCE(qr.points_awarded, 0)::double precision / qr.max_points) * 100.0
        ELSE NULL END
FROM latest la
INNER JOIN course.quiz_responses qr ON qr.attempt_id = la.id
WHERE qr.question_id = $4
`, courseID, quizItemID, userID, questionID).Scan(&ratio)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if ratio == nil || !isFiniteF64(*ratio) {
		return nil, nil
	}
	p := float32(*ratio)
	if p < 0 {
		p = 0
	} else if p > 100 {
		p = 100
	}
	return &p, nil
}

func isFiniteF64(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
