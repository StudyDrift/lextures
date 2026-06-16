package course

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SetSelfPacedSettings updates the self-paced configuration of a course (plan 15.2). Each
// argument is optional; nil leaves the existing value unchanged. Returns the updated course,
// or nil when the course code does not exist.
func SetSelfPacedSettings(ctx context.Context, pool *pgxpool.Pool, courseCode string, courseMode *string, openEnrollment, moduleGating *bool) (*CoursePublic, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.courses
SET course_mode = COALESCE($1::course.course_mode, course_mode),
    open_enrollment = COALESCE($2, open_enrollment),
    module_gating_enabled = COALESCE($3, module_gating_enabled),
    updated_at = NOW()
WHERE course_code = $4
`, courseMode, openEnrollment, moduleGating, courseCode)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetPublicByCourseCode(ctx, pool, courseCode)
}
