package enrollment

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/coursegrants"
)

// ListRosterRealtimeSubscriberUserIDs returns users who should receive roster realtime updates
// (enrolled members and anyone with enrollments:read on the course).
func ListRosterRealtimeSubscriberUserIDs(ctx context.Context, pool *pgxpool.Pool, courseCode string) ([]uuid.UUID, error) {
	readPerm := coursegrants.CourseEnrollmentsReadPermission(courseCode)
	rows, err := pool.Query(ctx, `
SELECT DISTINCT u.id
FROM (
  SELECT ce.user_id AS id
  FROM course.course_enrollments ce
  INNER JOIN course.courses c ON c.id = ce.course_id
  WHERE c.course_code = $1
    AND (ce.active OR ce.invitation_pending)
  UNION
  SELECT ucg.user_id AS id
  FROM course.user_course_grants ucg
  INNER JOIN course.courses c ON c.id = ucg.course_id
  WHERE c.course_code = $1
    AND ucg.permission_string = $2
) u
`, courseCode, readPerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}