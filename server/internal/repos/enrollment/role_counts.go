package enrollment

import (
	"context"
	"time"

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

// PeopleStub is a privacy-safe enrollment stub for checklist evidence (CC.1/CC.3).
type PeopleStub struct {
	UserID            uuid.UUID
	DisplayName       string
	Role              string
	Active            bool
	InvitationPending bool
	SectionID         *uuid.UUID
	CreatedAt         time.Time
}

// ListChecklistPeopleForCourse returns privacy-safe enrollment stubs enriched for
// CC.3 people rules (active, invitation pending, section, created_at).
// Directory-suppressed PII columns are never selected.
func ListChecklistPeopleForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]PeopleStub, error) {
	rows, err := pool.Query(ctx, `
SELECT ce.user_id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), 'Learner'),
       ce.role,
       ce.active,
       ce.invitation_pending,
       ce.section_id,
       ce.created_at
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
	var out []PeopleStub
	for rows.Next() {
		var row PeopleStub
		if err := rows.Scan(
			&row.UserID, &row.DisplayName, &row.Role,
			&row.Active, &row.InvitationPending, &row.SectionID, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
