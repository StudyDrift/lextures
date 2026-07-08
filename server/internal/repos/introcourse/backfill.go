package introcourse

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BackfillState is the singleton settings.intro_course_backfill row.
type BackfillState struct {
	StartedAt     *time.Time
	CompletedAt   *time.Time
	LastUserID    *uuid.UUID
	EnrolledCount int64
	UpdatedAt     time.Time
}

// LoadBackfillState returns the singleton backfill row (zero values when absent).
func LoadBackfillState(ctx context.Context, exec interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (BackfillState, error) {
	var st BackfillState
	err := exec.QueryRow(ctx, `
SELECT started_at, completed_at, last_user_id, enrolled_count, updated_at
FROM settings.intro_course_backfill
WHERE id = TRUE
`).Scan(&st.StartedAt, &st.CompletedAt, &st.LastUserID, &st.EnrolledCount, &st.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return BackfillState{}, nil
	}
	return st, err
}

// EnsureBackfillStarted marks started_at when the backfill begins.
func EnsureBackfillStarted(ctx context.Context, exec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, at time.Time) error {
	_, err := exec.Exec(ctx, `
UPDATE settings.intro_course_backfill
SET started_at = COALESCE(started_at, $1), updated_at = NOW()
WHERE id = TRUE
`, at)
	return err
}

// UpdateBackfillProgress persists the resume cursor and enrolled tally.
func UpdateBackfillProgress(ctx context.Context, exec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, lastUserID uuid.UUID, enrolledDelta int64) error {
	_, err := exec.Exec(ctx, `
UPDATE settings.intro_course_backfill
SET last_user_id = $1,
    enrolled_count = enrolled_count + $2,
    updated_at = NOW()
WHERE id = TRUE
`, lastUserID, enrolledDelta)
	return err
}

// MarkBackfillCompleted sets completed_at on the singleton row.
func MarkBackfillCompleted(ctx context.Context, exec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, at time.Time) error {
	_, err := exec.Exec(ctx, `
UPDATE settings.intro_course_backfill
SET completed_at = $1, updated_at = NOW()
WHERE id = TRUE
`, at)
	return err
}

// ClearBackfillCompleted clears completed_at so a backfill pass can run again (e.g. remaining users after an early complete).
func ClearBackfillCompleted(ctx context.Context, exec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}) error {
	_, err := exec.Exec(ctx, `
UPDATE settings.intro_course_backfill
SET completed_at = NULL, updated_at = NOW()
WHERE id = TRUE
`)
	return err
}

// CountBackfillRemaining returns eligible users not yet enrolled in the intro course.
func CountBackfillRemaining(ctx context.Context, pool *pgxpool.Pool, introCourseID, defaultOrgID uuid.UUID, skipParents bool) (int64, error) {
	parentClause := ""
	if skipParents {
		parentClause = ` AND u.account_type <> 'parent'`
	}
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::bigint
FROM "user".users u
WHERE u.account_type <> 'system'
  AND u.id <> $2
  AND u.org_id = $3`+parentClause+`
  AND NOT EXISTS (
      SELECT 1 FROM course.course_enrollments ce
      WHERE ce.course_id = $1 AND ce.user_id = u.id AND ce.role = 'student'
  )
`, introCourseID, SystemUserID, defaultOrgID).Scan(&n)
	return n, err
}