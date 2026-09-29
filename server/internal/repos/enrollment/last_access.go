package enrollment

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TouchLastCourseAccess updates last_course_access_at on the user's active enrollments
// for the course. Throttled to at most once per minute so repeated API calls are cheap.
// Returns true when at least one row was updated.
func TouchLastCourseAccess(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.course_enrollments
SET last_course_access_at = NOW()
WHERE course_id = $1
  AND user_id = $2
  AND active = true
  AND (
    last_course_access_at IS NULL
    OR last_course_access_at < NOW() - INTERVAL '1 minute'
  )
`, courseID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
