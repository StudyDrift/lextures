package contenttools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RosterRow is one enrolled learner's engagement with a tool instance (CT.4 FR-1/FR-2).
type RosterRow struct {
	EnrollmentID     uuid.UUID
	UserID           uuid.UUID
	DisplayName      string
	SectionID        *uuid.UUID
	Status           string
	ScoreRaw         *float64
	ScoreMax         *float64
	InteractionCount int
	LastInteractedAt *time.Time
	ResetCount       int
	HasState         bool
}

// RosterListParams filters and paginates an instance roster.
type RosterListParams struct {
	InstanceID uuid.UUID
	CourseID   uuid.UUID
	Status     string // empty = all; "not_started" includes learners without a state row
	SectionID  *uuid.UUID
	SectionIDs []uuid.UUID // TA narrowing; empty = no filter
	Page       int
	PageSize   int
}

// ListInstanceRoster returns one row per active student enrollment, left-joining state.
func ListInstanceRoster(ctx context.Context, pool *pgxpool.Pool, p RosterListParams) ([]RosterRow, int, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 50
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	offset := (p.Page - 1) * p.PageSize

	args := []any{p.CourseID, p.InstanceID}
	where := []string{
		`ce.course_id = $1`,
		`ce.role = 'student'`,
		`ce.active`,
	}
	n := 3
	if p.SectionID != nil {
		where = append(where, fmt.Sprintf(`ce.section_id = $%d`, n))
		args = append(args, *p.SectionID)
		n++
	}
	if len(p.SectionIDs) > 0 {
		where = append(where, fmt.Sprintf(`ce.section_id = ANY($%d::uuid[])`, n))
		args = append(args, p.SectionIDs)
		n++
	}
	statusFilter := strings.TrimSpace(strings.ToLower(p.Status))
	switch statusFilter {
	case "", "all":
		// no filter
	case "not_started":
		where = append(where, `(s.id IS NULL OR s.status = 'not_started')`)
	case "in_progress", "submitted", "completed":
		where = append(where, fmt.Sprintf(`s.status = $%d`, n))
		args = append(args, statusFilter)
		n++
	default:
		return nil, 0, fmt.Errorf("invalid status filter")
	}

	whereSQL := strings.Join(where, " AND ")
	countQ := `
SELECT COUNT(*)::int
FROM course.course_enrollments ce
LEFT JOIN course.content_tool_states s
  ON s.enrollment_id = ce.id AND s.instance_id = $2 AND s.scope = 'enrollment'
WHERE ` + whereSQL
	var total int
	if err := pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, p.PageSize, offset)
	listQ := fmt.Sprintf(`
SELECT
  ce.id,
  ce.user_id,
  COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) AS display_name,
  ce.section_id,
  COALESCE(s.status, 'not_started') AS status,
  s.score_raw,
  s.score_max,
  COALESCE(s.interaction_count, 0) AS interaction_count,
  s.last_interacted_at,
  COALESCE(s.reset_count, 0) AS reset_count,
  (s.id IS NOT NULL) AS has_state
FROM course.course_enrollments ce
INNER JOIN "user".users u ON u.id = ce.user_id
LEFT JOIN course.content_tool_states s
  ON s.enrollment_id = ce.id AND s.instance_id = $2 AND s.scope = 'enrollment'
WHERE %s
ORDER BY display_name ASC, ce.id ASC
LIMIT $%d OFFSET $%d
`, whereSQL, n, n+1)

	rows, err := pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]RosterRow, 0, p.PageSize)
	for rows.Next() {
		var r RosterRow
		if err := rows.Scan(
			&r.EnrollmentID, &r.UserID, &r.DisplayName, &r.SectionID,
			&r.Status, &r.ScoreRaw, &r.ScoreMax, &r.InteractionCount,
			&r.LastInteractedAt, &r.ResetCount, &r.HasState,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// GetInstanceStateDetail returns full state for one enrollment (nil if not started).
func GetInstanceStateDetail(ctx context.Context, pool *pgxpool.Pool, instanceID, enrollmentID uuid.UUID) (*StateRow, error) {
	return GetState(ctx, pool, instanceID, enrollmentID)
}

// EnrollmentDisplayName returns a display label for an enrollment in a course.
func EnrollmentDisplayName(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (string, uuid.UUID, error) {
	var name string
	var userID uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT ce.user_id, COALESCE(NULLIF(TRIM(u.display_name), ''), u.email)
FROM course.course_enrollments ce
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE ce.id = $1
`, enrollmentID).Scan(&userID, &name)
	return name, userID, err
}
