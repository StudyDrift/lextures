package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetArchived toggles course archived flag and returns updated public row.
// When archived is true, archivedBy records who archived the course; when false, metadata is cleared.
func SetArchived(ctx context.Context, pool *pgxpool.Pool, courseCode string, archived bool, archivedBy *uuid.UUID) (*CoursePublic, error) {
	var q string
	var args []any
	if archived {
		q = `
			UPDATE course.courses
			SET
				archived = TRUE,
				archived_at = NOW(),
				archived_by_user_id = $1,
				updated_at = NOW()
			WHERE course_code = $2
		`
		args = []any{archivedBy, courseCode}
	} else {
		q = `
			UPDATE course.courses
			SET
				archived = FALSE,
				archived_at = NULL,
				archived_by_user_id = NULL,
				updated_at = NOW()
			WHERE course_code = $1
		`
		args = []any{courseCode}
	}
	tag, err := pool.Exec(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetPublicByCourseCode(ctx, pool, courseCode)
}

