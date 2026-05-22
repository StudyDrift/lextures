// Package studentprogress loads cached and live per-enrollment progress data (plan 9.1).
package studentprogress

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const refreshStaleAfter = 5 * time.Minute

// SnapshotRow is one row from analytics.student_progress.
type SnapshotRow struct {
	EnrollmentID         uuid.UUID
	CourseID             uuid.UUID
	UserID               uuid.UUID
	AssignmentsSubmitted int
	AssignmentsTotal     int
	AvgQuizScore         *float32
	ModuleViewsCount     int
	ModulesTotal         int
	LastActiveAt         *time.Time
}

// RefreshMeta holds materialized view refresh timing.
type RefreshMeta struct {
	RefreshedAt time.Time
}

// GetRefreshMeta returns when the progress view was last refreshed.
func GetRefreshMeta(ctx context.Context, pool *pgxpool.Pool) (RefreshMeta, error) {
	var t time.Time
	err := pool.QueryRow(ctx, `SELECT refreshed_at FROM analytics.student_progress_refresh WHERE id = 1`).Scan(&t)
	if err != nil {
		return RefreshMeta{}, err
	}
	return RefreshMeta{RefreshedAt: t.UTC()}, nil
}

// RefreshViewIfStale runs REFRESH MATERIALIZED VIEW CONCURRENTLY when older than refreshStaleAfter.
func RefreshViewIfStale(ctx context.Context, pool *pgxpool.Pool) (refreshed bool, err error) {
	meta, err := GetRefreshMeta(ctx, pool)
	if err != nil {
		return false, err
	}
	if time.Since(meta.RefreshedAt) < refreshStaleAfter {
		return false, nil
	}
	return true, RefreshViewNow(ctx, pool)
}

// RefreshViewNow refreshes the materialized view and updates metadata.
func RefreshViewNow(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.student_progress`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE analytics.student_progress_refresh SET refreshed_at = NOW() WHERE id = 1`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetSnapshot returns cached progress for an enrollment or nil.
func GetSnapshot(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (*SnapshotRow, error) {
	const q = `
SELECT enrollment_id, course_id, user_id,
       assignments_submitted, assignments_total, avg_quiz_score,
       module_views_count, modules_total, last_active_at
FROM analytics.student_progress
WHERE enrollment_id = $1`
	var r SnapshotRow
	var avg *float32
	err := pool.QueryRow(ctx, q, enrollmentID).Scan(
		&r.EnrollmentID, &r.CourseID, &r.UserID,
		&r.AssignmentsSubmitted, &r.AssignmentsTotal, &avg,
		&r.ModuleViewsCount, &r.ModulesTotal, &r.LastActiveAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if avg != nil {
		r.AvgQuizScore = avg
	}
	return &r, nil
}

// AvgGradePercent computes mean percent across posted course grades for gradable items.
func AvgGradePercent(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (*float64, error) {
	var avg *float64
	err := pool.QueryRow(ctx, `
SELECT AVG(
    CASE
        WHEN COALESCE(ma.points_worth, mq.points_worth, 0) > 0
            THEN (cg.points_earned / COALESCE(ma.points_worth, mq.points_worth)::float8) * 100.0
        ELSE NULL
    END
)::float8
FROM course.course_grades cg
INNER JOIN course.course_structure_items csi ON csi.id = cg.module_item_id
LEFT JOIN course.module_assignments ma ON ma.structure_item_id = csi.id
LEFT JOIN course.module_quizzes mq ON mq.structure_item_id = csi.id
WHERE cg.course_id = $1 AND cg.student_user_id = $2
  AND csi.published AND NOT csi.archived
  AND csi.kind IN ('assignment', 'quiz')
`, courseID, userID).Scan(&avg)
	if err != nil {
		return nil, err
	}
	if avg != nil && math.IsNaN(*avg) {
		return nil, nil
	}
	return avg, nil
}

// MissingItemRow is overdue unsubmitted work.
type MissingItemRow struct {
	ItemID      uuid.UUID
	Title       string
	Kind        string
	DueAt       *time.Time
	DaysOverdue int
	GradeStatus string
}

// ListMissing returns published items past due without submission/attempt.
func ListMissing(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID, now time.Time) ([]MissingItemRow, error) {
	rows, err := pool.Query(ctx, `
SELECT csi.id, csi.title, csi.kind, csi.due_at,
       GREATEST(0, EXTRACT(DAY FROM ($3::timestamptz - csi.due_at))::int) AS days_overdue,
       COALESCE(
           CASE
               WHEN csi.kind = 'assignment' AND mas.id IS NOT NULL THEN 'submitted'
               WHEN csi.kind = 'quiz' AND qa.id IS NOT NULL THEN 'attempted'
               ELSE 'missing'
           END,
           'missing'
       ) AS grade_status
FROM course.course_structure_items csi
LEFT JOIN course.module_assignment_submissions mas
    ON mas.module_item_id = csi.id AND mas.submitted_by = $2
LEFT JOIN LATERAL (
    SELECT qa.id
    FROM course.quiz_attempts qa
    WHERE qa.structure_item_id = csi.id AND qa.student_user_id = $2 AND qa.status = 'submitted'
    ORDER BY qa.submitted_at DESC NULLS LAST
    LIMIT 1
) qa ON csi.kind = 'quiz'
WHERE csi.course_id = $1
  AND csi.published AND NOT csi.archived
  AND csi.kind IN ('assignment', 'quiz')
  AND csi.due_at IS NOT NULL
  AND csi.due_at < $3
  AND (
      (csi.kind = 'assignment' AND mas.id IS NULL)
      OR (csi.kind = 'quiz' AND qa.id IS NULL)
  )
ORDER BY csi.due_at ASC
`, courseID, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MissingItemRow
	for rows.Next() {
		var r MissingItemRow
		if err := rows.Scan(&r.ItemID, &r.Title, &r.Kind, &r.DueAt, &r.DaysOverdue, &r.GradeStatus); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AssignmentRow is assignment progress detail.
type AssignmentRow struct {
	ItemID      uuid.UUID
	Title       string
	DueAt       *time.Time
	SubmittedAt *time.Time
	Points      *float64
	PointsWorth *int
}

// ListAssignments returns all published assignments with submission/grade info.
func ListAssignments(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) ([]AssignmentRow, error) {
	rows, err := pool.Query(ctx, `
SELECT csi.id, csi.title, csi.due_at, mas.submitted_at, cg.points_earned, ma.points_worth
FROM course.course_structure_items csi
INNER JOIN course.module_assignments ma ON ma.structure_item_id = csi.id
LEFT JOIN course.module_assignment_submissions mas
    ON mas.module_item_id = csi.id AND mas.submitted_by = $2
LEFT JOIN course.course_grades cg
    ON cg.module_item_id = csi.id AND cg.student_user_id = $2
WHERE csi.course_id = $1 AND csi.kind = 'assignment' AND csi.published AND NOT csi.archived
ORDER BY csi.due_at NULLS LAST, csi.sort_order ASC
`, courseID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AssignmentRow
	for rows.Next() {
		var r AssignmentRow
		if err := rows.Scan(&r.ItemID, &r.Title, &r.DueAt, &r.SubmittedAt, &r.Points, &r.PointsWorth); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// QuizRow is quiz attempt detail.
type QuizRow struct {
	AttemptID    uuid.UUID
	ItemID       uuid.UUID
	Title        string
	SubmittedAt  time.Time
	ScorePercent *float32
}

// ListQuizAttempts returns submitted quiz attempts for the student.
func ListQuizAttempts(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) ([]QuizRow, error) {
	rows, err := pool.Query(ctx, `
SELECT qa.id, csi.id, csi.title, qa.submitted_at, qa.score_percent
FROM course.quiz_attempts qa
INNER JOIN course.course_structure_items csi ON csi.id = qa.structure_item_id
WHERE qa.course_id = $1 AND qa.student_user_id = $2 AND qa.status = 'submitted'
ORDER BY qa.submitted_at ASC
`, courseID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QuizRow
	for rows.Next() {
		var r QuizRow
		if err := rows.Scan(&r.AttemptID, &r.ItemID, &r.Title, &r.SubmittedAt, &r.ScorePercent); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ActivityRow is one timeline event.
type ActivityRow struct {
	OccurredAt time.Time
	Kind       string
	Label      string
	Detail     string
}

// ListActivity returns paginated activity (newest first). Cursor is RFC3339 occurred_at.
func ListActivity(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID, cursor *time.Time, limit int) ([]ActivityRow, *time.Time, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	args := []any{courseID, userID, limit + 1}
	cursorClause := ""
	if cursor != nil {
		cursorClause = "AND occurred_at < $4"
		args = append(args, *cursor)
	}
	q := fmt.Sprintf(`
SELECT occurred_at, kind, label, COALESCE(detail, '') FROM (
    SELECT mas.submitted_at AS occurred_at, 'submission' AS kind,
           csi.title AS label, 'Assignment submitted' AS detail
    FROM course.module_assignment_submissions mas
    INNER JOIN course.course_structure_items csi ON csi.id = mas.module_item_id
    WHERE mas.course_id = $1 AND mas.submitted_by = $2
    UNION ALL
    SELECT qa.submitted_at, 'quiz',
           csi.title, 'Quiz submitted: ' || COALESCE(ROUND(qa.score_percent::numeric, 1)::text, '—') || '%%'
    FROM course.quiz_attempts qa
    INNER JOIN course.course_structure_items csi ON csi.id = qa.structure_item_id
    WHERE qa.course_id = $1 AND qa.student_user_id = $2 AND qa.status = 'submitted' AND qa.submitted_at IS NOT NULL
    UNION ALL
    SELECT ua.occurred_at, 'view',
           COALESCE(csi.title, 'Course'), 'Content viewed'
    FROM "user".user_audit ua
    LEFT JOIN course.course_structure_items csi ON csi.id = ua.structure_item_id
    WHERE ua.course_id = $1 AND ua.user_id = $2 AND ua.event_kind = 'content_open'
) events
WHERE occurred_at IS NOT NULL %s
ORDER BY occurred_at DESC
LIMIT $3
`, cursorClause)
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []ActivityRow
	for rows.Next() {
		var r ActivityRow
		if err := rows.Scan(&r.OccurredAt, &r.Kind, &r.Label, &r.Detail); err != nil {
			return nil, nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var next *time.Time
	if len(out) > limit {
		t := out[limit].OccurredAt.UTC()
		next = &t
		out = out[:limit]
	}
	return out, next, nil
}

// NoteRow is an instructor note.
type NoteRow struct {
	ID        uuid.UUID
	AuthorID  uuid.UUID
	NoteText  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListNotes returns notes for an enrollment.
func ListNotes(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) ([]NoteRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, author_id, note_text, created_at, updated_at
FROM analytics.instructor_progress_notes
WHERE enrollment_id = $1
ORDER BY updated_at DESC
`, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NoteRow
	for rows.Next() {
		var r NoteRow
		if err := rows.Scan(&r.ID, &r.AuthorID, &r.NoteText, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateNote inserts a note.
func CreateNote(ctx context.Context, pool *pgxpool.Pool, enrollmentID, authorID uuid.UUID, text string) (NoteRow, error) {
	var r NoteRow
	err := pool.QueryRow(ctx, `
INSERT INTO analytics.instructor_progress_notes (enrollment_id, author_id, note_text)
VALUES ($1, $2, $3)
RETURNING id, author_id, note_text, created_at, updated_at
`, enrollmentID, authorID, text).Scan(&r.ID, &r.AuthorID, &r.NoteText, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

// UpdateNote updates note text when author matches.
func UpdateNote(ctx context.Context, pool *pgxpool.Pool, noteID, authorID uuid.UUID, text string) (NoteRow, error) {
	var r NoteRow
	err := pool.QueryRow(ctx, `
UPDATE analytics.instructor_progress_notes
SET note_text = $3, updated_at = NOW()
WHERE id = $1 AND author_id = $2
RETURNING id, author_id, note_text, created_at, updated_at
`, noteID, authorID, text).Scan(&r.ID, &r.AuthorID, &r.NoteText, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return NoteRow{}, pgx.ErrNoRows
	}
	return r, err
}

// DeleteNote removes a note when author matches.
func DeleteNote(ctx context.Context, pool *pgxpool.Pool, noteID, authorID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM analytics.instructor_progress_notes WHERE id = $1 AND author_id = $2
`, noteID, authorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// Pct computes a safe percentage 0–100.
func Pct(n, total int) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round((float64(n)/float64(total))*1000) / 10
}
