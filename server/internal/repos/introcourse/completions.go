package introcourse

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// CompletionRow is one settings.intro_course_completions record.
type CompletionRow struct {
	UserID       uuid.UUID
	CompletedAt  time.Time
	FinalGrade   *float64
	CredentialID *uuid.UUID
	EventSent    bool
	CreatedAt    time.Time
}

// GetCompletion returns the completion row for userID, or nil when absent.
func GetCompletion(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, userID uuid.UUID) (*CompletionRow, error) {
	var row CompletionRow
	err := q.QueryRow(ctx, `
SELECT user_id, completed_at, final_grade, credential_id, event_sent, created_at
FROM settings.intro_course_completions
WHERE user_id = $1
`, userID).Scan(
		&row.UserID, &row.CompletedAt, &row.FinalGrade, &row.CredentialID, &row.EventSent, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// InsertCompletionIfAbsent records completion once per user (idempotent).
func InsertCompletionIfAbsent(ctx context.Context, tx pgx.Tx, userID uuid.UUID, finalGrade *float64) (bool, *CompletionRow, error) {
	var row CompletionRow
	err := tx.QueryRow(ctx, `
INSERT INTO settings.intro_course_completions (user_id, final_grade)
VALUES ($1, $2)
ON CONFLICT (user_id) DO NOTHING
RETURNING user_id, completed_at, final_grade, credential_id, event_sent, created_at
`, userID, finalGrade).Scan(
		&row.UserID, &row.CompletedAt, &row.FinalGrade, &row.CredentialID, &row.EventSent, &row.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, gerr := GetCompletion(ctx, tx, userID)
		return false, existing, gerr
	}
	if err != nil {
		return false, nil, err
	}
	return true, &row, nil
}

// SetCredentialID links an issued credential to the completion row.
func SetCredentialID(ctx context.Context, q interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}, userID, credentialID uuid.UUID) error {
	_, err := q.Exec(ctx, `
UPDATE settings.intro_course_completions
SET credential_id = $2
WHERE user_id = $1 AND credential_id IS NULL
`, userID, credentialID)
	return err
}

// MarkEventSent sets event_sent=true once (idempotent).
func MarkEventSent(ctx context.Context, q interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}, userID uuid.UUID) error {
	_, err := q.Exec(ctx, `
UPDATE settings.intro_course_completions
SET event_sent = TRUE
WHERE user_id = $1 AND NOT event_sent
`, userID)
	return err
}

// HasCompleted reports whether userID has a completion row.
func HasCompleted(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, userID uuid.UUID) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM settings.intro_course_completions WHERE user_id = $1)
`, userID).Scan(&ok)
	return ok, err
}

// CountCompleted returns learners with a completion row.
func CountCompleted(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT COUNT(*)::int FROM settings.intro_course_completions`).Scan(&n)
	return n, err
}

// CountEnrolledStudents returns active student enrollments in courseID.
func CountEnrolledStudents(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, courseID uuid.UUID) (int, error) {
	var n int
	err := q.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM course.course_enrollments
WHERE course_id = $1 AND role = 'student' AND active AND state = 'active'
`, courseID).Scan(&n)
	return n, err
}

// ModuleFunnelRow is per-module quiz-attempt counts for admin analytics.
type ModuleFunnelRow struct {
	ModuleSlug    string
	ModuleTitle   string
	SortOrder     int
	QuizAttempted int
}

// ListModuleFunnel returns quiz-attempt counts per module for enrolled students.
func ListModuleFunnel(ctx context.Context, q interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, courseID uuid.UUID) ([]ModuleFunnelRow, error) {
	rows, err := q.Query(ctx, `
SELECT parent.slug, parent.title, parent.sort_order,
       COUNT(DISTINCT qa.student_user_id)::int AS quiz_attempted
FROM (
    SELECT mod_ici.slug, mod_csi.title, mod_csi.sort_order, quiz_csi.id AS quiz_item_id
    FROM settings.intro_course_items quiz_ici
    INNER JOIN course.course_structure_items quiz_csi ON quiz_csi.id = quiz_ici.structure_item_id
    INNER JOIN course.course_structure_items mod_csi ON mod_csi.id = quiz_csi.parent_id
    INNER JOIN settings.intro_course_items mod_ici ON mod_ici.structure_item_id = mod_csi.id
    WHERE quiz_csi.course_id = $1
      AND quiz_csi.kind = 'quiz' AND quiz_csi.published AND NOT quiz_csi.archived
      AND mod_csi.kind = 'module' AND mod_csi.published AND NOT mod_csi.archived
      AND quiz_ici.slug LIKE '%.knowledge-check'
) parent
LEFT JOIN course.quiz_attempts qa
    ON qa.structure_item_id = parent.quiz_item_id
   AND qa.course_id = $1
   AND qa.status = 'submitted'
   AND qa.student_user_id IN (
       SELECT user_id FROM course.course_enrollments
       WHERE course_id = $1 AND role = 'student' AND active AND state = 'active'
   )
GROUP BY parent.slug, parent.title, parent.sort_order
ORDER BY parent.sort_order
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModuleFunnelRow
	for rows.Next() {
		var r ModuleFunnelRow
		if err := rows.Scan(&r.ModuleSlug, &r.ModuleTitle, &r.SortOrder, &r.QuizAttempted); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteCompletionByUser removes a completion row (GDPR erase).
func DeleteCompletionByUser(ctx context.Context, q interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}, userID uuid.UUID) error {
	_, err := q.Exec(ctx, `DELETE FROM settings.intro_course_completions WHERE user_id = $1`, userID)
	return err
}

// ListEnrolledStudentIDs returns active student user ids for courseID.
func ListEnrolledStudentIDs(ctx context.Context, q interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}, courseID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.Query(ctx, `
SELECT user_id FROM course.course_enrollments
WHERE course_id = $1 AND role = 'student' AND active AND state = 'active'
ORDER BY user_id
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}