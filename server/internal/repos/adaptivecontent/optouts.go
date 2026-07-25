package adaptivecontent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OptoutRow is a course.adaptive_content_optouts row (AC.6).
type OptoutRow struct {
	CourseID  uuid.UUID
	UserID    uuid.UUID
	OptedOut  bool
	UpdatedAt time.Time
}

// GetOptout returns the opt-out row for (course, user), or nil when no row exists.
// A missing row means the student has not opted out (optedOut=false).
func GetOptout(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (*OptoutRow, error) {
	var r OptoutRow
	err := pool.QueryRow(ctx, `
SELECT course_id, user_id, opted_out, updated_at
FROM course.adaptive_content_optouts
WHERE course_id = $1 AND user_id = $2
`, courseID, userID).Scan(&r.CourseID, &r.UserID, &r.OptedOut, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// IsOptedOut reports whether the student has an active opt-out for the course.
func IsOptedOut(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (bool, error) {
	row, err := GetOptout(ctx, pool, courseID, userID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}
	return row.OptedOut, nil
}

// SetOptout upserts the student's opt-out preference for a course.
func SetOptout(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID, optedOut bool) (*OptoutRow, error) {
	var r OptoutRow
	err := pool.QueryRow(ctx, `
INSERT INTO course.adaptive_content_optouts (course_id, user_id, opted_out, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (course_id, user_id) DO UPDATE SET
  opted_out = EXCLUDED.opted_out,
  updated_at = NOW()
RETURNING course_id, user_id, opted_out, updated_at
`, courseID, userID, optedOut).Scan(&r.CourseID, &r.UserID, &r.OptedOut, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
