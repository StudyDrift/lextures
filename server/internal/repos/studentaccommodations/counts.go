package studentaccommodations

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CountActiveForCourse returns the number of active accommodations that apply
// to the course (course-scoped or global). Read-only helper for CC.1.
func CountActiveForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM course.student_accommodations sa
WHERE sa.active = true
  AND (sa.course_id = $1 OR sa.course_id IS NULL)
`, courseID).Scan(&n)
	return n, err
}
