package contenttools

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EnrollmentStateRow is one enrollment's state plus role/section for aggregates (CT.21).
type EnrollmentStateRow struct {
	EnrollmentID uuid.UUID
	Role         string
	SectionID    *uuid.UUID
	StateJSON    json.RawMessage
}

// ListEnrollmentStatesForAggregate returns enrollment-scoped states with role and section.
func ListEnrollmentStatesForAggregate(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID) ([]EnrollmentStateRow, error) {
	rows, err := pool.Query(ctx, `
SELECT cts.enrollment_id, COALESCE(ce.role, ''), ce.section_id, cts.state_json
FROM course.content_tool_states cts
JOIN course.course_enrollments ce ON ce.id = cts.enrollment_id
WHERE cts.instance_id = $1 AND cts.scope = $2
ORDER BY cts.updated_at DESC
`, instanceID, ScopeEnrollment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EnrollmentStateRow
	for rows.Next() {
		var r EnrollmentStateRow
		var raw []byte
		if err := rows.Scan(&r.EnrollmentID, &r.Role, &r.SectionID, &raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			raw = []byte(`{}`)
		}
		r.StateJSON = json.RawMessage(raw)
		out = append(out, r)
	}
	return out, rows.Err()
}
