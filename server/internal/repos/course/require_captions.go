package course

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetRequireCaptions updates the require_captions flag for a course (plan 12.4).
func SetRequireCaptions(ctx context.Context, pool *pgxpool.Pool, courseCode string, require bool) (*CoursePublic, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE course.courses SET require_captions = $1, updated_at = NOW() WHERE course_code = $2`,
		require, courseCode,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetPublicByCourseCode(ctx, pool, courseCode)
}
