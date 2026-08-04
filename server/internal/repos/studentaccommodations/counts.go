package studentaccommodations

import (
	"context"
	"time"

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

// TypeAggregate is a privacy-safe count of active accommodations of one type.
// Never includes user identifiers (CC.5 FR-21 / AC-10).
type TypeAggregate struct {
	Type  string
	Count int
}

// AggregateActiveTypesForCourse returns counts by accommodation type for the course
// (course-scoped or global). Types with zero count are omitted.
func AggregateActiveTypesForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]TypeAggregate, *time.Time, error) {
	rows, err := pool.Query(ctx, `
SELECT typ, COUNT(*)::int AS n, MAX(created_at) AS latest
FROM (
  SELECT 'extended_time' AS typ, created_at
  FROM course.student_accommodations
  WHERE active = true AND (course_id = $1 OR course_id IS NULL) AND time_multiplier > 1.0
  UNION ALL
  SELECT 'extra_attempts', created_at
  FROM course.student_accommodations
  WHERE active = true AND (course_id = $1 OR course_id IS NULL) AND extra_attempts > 0
  UNION ALL
  SELECT 'reduced_distraction', created_at
  FROM course.student_accommodations
  WHERE active = true AND (course_id = $1 OR course_id IS NULL) AND reduced_distraction_mode
  UNION ALL
  SELECT 'separate_setting', created_at
  FROM course.student_accommodations
  WHERE active = true AND (course_id = $1 OR course_id IS NULL) AND separate_setting
) t
GROUP BY typ
ORDER BY typ
`, courseID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []TypeAggregate
	var latest *time.Time
	for rows.Next() {
		var typ string
		var n int
		var created time.Time
		if err := rows.Scan(&typ, &n, &created); err != nil {
			return nil, nil, err
		}
		out = append(out, TypeAggregate{Type: typ, Count: n})
		if latest == nil || created.After(*latest) {
			t := created
			latest = &t
		}
	}
	return out, latest, rows.Err()
}
