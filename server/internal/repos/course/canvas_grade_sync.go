package course

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetCanvasGradeSyncEnabled toggles whether grades are pushed back to Canvas when saved in Lextures.
func SetCanvasGradeSyncEnabled(ctx context.Context, pool *pgxpool.Pool, courseCode string, enabled bool) (*CoursePublic, error) {
	tag, err := pool.Exec(ctx, `
		UPDATE course.courses
		SET canvas_grade_sync_enabled = $1, updated_at = NOW()
		WHERE course_code = $2
	`, enabled, courseCode)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetPublicByCourseCode(ctx, pool, courseCode)
}