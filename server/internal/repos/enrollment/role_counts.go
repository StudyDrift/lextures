package enrollment

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CountByRoleForCourse returns active/invited enrollment counts keyed by role
// for a course. Used by the course checklist snapshot loader (CC.1) — read-only.
func CountByRoleForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[string]int, error) {
	rows, err := pool.Query(ctx, `
SELECT ce.role, COUNT(*)::int
FROM course.course_enrollments ce
WHERE ce.course_id = $1
  AND (ce.active OR ce.invitation_pending OR ce.state IN ('withdrawn', 'dropped', 'no_credit', 'audit', 'incomplete'))
GROUP BY ce.role
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var role string
		var n int
		if err := rows.Scan(&role, &n); err != nil {
			return nil, err
		}
		out[role] = n
	}
	return out, rows.Err()
}

// ListPeopleStubsForCourse returns privacy-safe enrollment stubs (opaque user id +
// display name + role). Directory-suppressed PII columns are never selected (CC.1).
func ListPeopleStubsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]struct {
	UserID      uuid.UUID
	DisplayName string
	Role        string
}, error) {
	rows, err := pool.Query(ctx, `
SELECT ce.user_id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), 'Learner'),
       ce.role
FROM course.course_enrollments ce
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE ce.course_id = $1
  AND (ce.active OR ce.invitation_pending)
ORDER BY ce.role, COALESCE(NULLIF(TRIM(u.display_name), ''), 'Learner')
LIMIT 500
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		UserID      uuid.UUID
		DisplayName string
		Role        string
	}
	for rows.Next() {
		var row struct {
			UserID      uuid.UUID
			DisplayName string
			Role        string
		}
		if err := rows.Scan(&row.UserID, &row.DisplayName, &row.Role); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
