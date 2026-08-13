package contenttools

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StateSummaryRow is analytics.content_tool_state_summaries.
type StateSummaryRow struct {
	StateID            uuid.UUID
	InstanceID         uuid.UUID
	CourseID           uuid.UUID
	EnrollmentID       uuid.UUID
	ToolID             string
	Role               string
	Engaged            bool
	Completed          bool
	ScorePct           *float64
	DurationMs         *int
	FacetsJSON         json.RawMessage
	ProjectionVersion  int
	UpdatedAt          time.Time
	DisplayName        string // joined when listing for instructors
	Status             string // joined from state when available
}

// UpsertStateSummary inserts or updates the summary for one state row.
func UpsertStateSummary(ctx context.Context, pool *pgxpool.Pool, row StateSummaryRow) error {
	if pool == nil {
		return errors.New("nil pool")
	}
	facets := row.FacetsJSON
	if len(facets) == 0 {
		facets = json.RawMessage(`{}`)
	}
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.content_tool_state_summaries (
  state_id, instance_id, course_id, enrollment_id, tool_id, role,
  engaged, completed, score_pct, duration_ms, facets_json, projection_version, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,NOW())
ON CONFLICT (state_id) DO UPDATE SET
  instance_id = EXCLUDED.instance_id,
  course_id = EXCLUDED.course_id,
  enrollment_id = EXCLUDED.enrollment_id,
  tool_id = EXCLUDED.tool_id,
  role = EXCLUDED.role,
  engaged = EXCLUDED.engaged,
  completed = EXCLUDED.completed,
  score_pct = EXCLUDED.score_pct,
  duration_ms = EXCLUDED.duration_ms,
  facets_json = EXCLUDED.facets_json,
  projection_version = EXCLUDED.projection_version,
  updated_at = NOW()
`, row.StateID, row.InstanceID, row.CourseID, row.EnrollmentID, row.ToolID, row.Role,
		row.Engaged, row.Completed, row.ScorePct, row.DurationMs, facets, row.ProjectionVersion)
	return err
}

// ResetStateSummary marks a summary as not engaged/completed after a CT.4 reset.
func ResetStateSummary(ctx context.Context, pool *pgxpool.Pool, stateID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE analytics.content_tool_state_summaries
SET engaged = FALSE,
    completed = FALSE,
    score_pct = NULL,
    duration_ms = NULL,
    facets_json = '{}'::jsonb,
    updated_at = NOW()
WHERE state_id = $1
`, stateID)
	return err
}

// ListSummariesForInstance returns summaries with display names for aggregation.
func ListSummariesForInstance(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID) ([]StateSummaryRow, error) {
	rows, err := pool.Query(ctx, `
SELECT s.state_id, s.instance_id, s.course_id, s.enrollment_id, s.tool_id, s.role,
       s.engaged, s.completed, s.score_pct, s.duration_ms, s.facets_json, s.projection_version, s.updated_at,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email, ''),
       COALESCE(st.status, 'not_started')
FROM analytics.content_tool_state_summaries s
INNER JOIN course.course_enrollments ce ON ce.id = s.enrollment_id
INNER JOIN "user".users u ON u.id = ce.user_id
LEFT JOIN course.content_tool_states st ON st.id = s.state_id
WHERE s.instance_id = $1
`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSummaryRows(rows)
}

// ListSummariesForItem returns summaries for all instances on a structure item.
func ListSummariesForItem(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) ([]StateSummaryRow, error) {
	rows, err := pool.Query(ctx, `
SELECT s.state_id, s.instance_id, s.course_id, s.enrollment_id, s.tool_id, s.role,
       s.engaged, s.completed, s.score_pct, s.duration_ms, s.facets_json, s.projection_version, s.updated_at,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email, ''),
       COALESCE(st.status, 'not_started')
FROM analytics.content_tool_state_summaries s
INNER JOIN course.content_tool_instances i ON i.id = s.instance_id
INNER JOIN course.course_enrollments ce ON ce.id = s.enrollment_id
INNER JOIN "user".users u ON u.id = ce.user_id
LEFT JOIN course.content_tool_states st ON st.id = s.state_id
WHERE s.course_id = $1 AND i.structure_item_id = $2 AND i.status = 'active'
`, courseID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSummaryRows(rows)
}

// ListSummariesForCourse returns all summaries in a course.
func ListSummariesForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]StateSummaryRow, error) {
	rows, err := pool.Query(ctx, `
SELECT s.state_id, s.instance_id, s.course_id, s.enrollment_id, s.tool_id, s.role,
       s.engaged, s.completed, s.score_pct, s.duration_ms, s.facets_json, s.projection_version, s.updated_at,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email, ''),
       COALESCE(st.status, 'not_started')
FROM analytics.content_tool_state_summaries s
INNER JOIN course.course_enrollments ce ON ce.id = s.enrollment_id
INNER JOIN "user".users u ON u.id = ce.user_id
LEFT JOIN course.content_tool_states st ON st.id = s.state_id
WHERE s.course_id = $1
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSummaryRows(rows)
}

// ListSummariesForEnrollment returns the caller's summaries for instances (student progress).
func ListSummariesForEnrollment(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, instanceIDs []uuid.UUID) ([]StateSummaryRow, error) {
	if len(instanceIDs) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT s.state_id, s.instance_id, s.course_id, s.enrollment_id, s.tool_id, s.role,
       s.engaged, s.completed, s.score_pct, s.duration_ms, s.facets_json, s.projection_version, s.updated_at,
       '', COALESCE(st.status, 'not_started')
FROM analytics.content_tool_state_summaries s
LEFT JOIN course.content_tool_states st ON st.id = s.state_id
WHERE s.enrollment_id = $1 AND s.instance_id = ANY($2::uuid[])
`, enrollmentID, instanceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSummaryRows(rows)
}

// EnrollmentRole returns the enrollment role string for summary writes.
func EnrollmentRole(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (string, error) {
	var role string
	err := pool.QueryRow(ctx, `SELECT role FROM course.course_enrollments WHERE id = $1`, enrollmentID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return role, err
}

// ListStatesNeedingSummary returns enrollment-scoped states missing a summary or with old projection version.
func ListStatesNeedingSummary(ctx context.Context, pool *pgxpool.Pool, toolID string, projectionVersion int, limit int) ([]StateRow, error) {
	if limit <= 0 {
		limit = 500
	}
	q := `
SELECT ` + stateColsAliased + `
FROM course.content_tool_states st
INNER JOIN course.content_tool_instances i ON i.id = st.instance_id
LEFT JOIN analytics.content_tool_state_summaries s ON s.state_id = st.id
WHERE st.scope = 'enrollment'
  AND ($1 = '' OR i.tool_id = $1)
  AND (s.state_id IS NULL OR s.projection_version < $2)
ORDER BY st.updated_at ASC
LIMIT $3
`
	rows, err := pool.Query(ctx, q, toolID, projectionVersion, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]StateRow, 0)
	for rows.Next() {
		st, err := scanState(rows)
		if err != nil {
			return nil, err
		}
		if st != nil {
			out = append(out, *st)
		}
	}
	return out, rows.Err()
}

// InstanceCourseAndItem resolves course_id and optional structure_item_id for cache invalidation.
func InstanceCourseAndItem(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID) (courseID uuid.UUID, itemID *uuid.UUID, toolID string, err error) {
	err = pool.QueryRow(ctx, `
SELECT course_id, structure_item_id, tool_id FROM course.content_tool_instances WHERE id = $1
`, instanceID).Scan(&courseID, &itemID, &toolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil, "", nil
	}
	return courseID, itemID, toolID, err
}

func scanSummaryRows(rows pgx.Rows) ([]StateSummaryRow, error) {
	out := make([]StateSummaryRow, 0)
	for rows.Next() {
		var r StateSummaryRow
		var facets []byte
		if err := rows.Scan(
			&r.StateID, &r.InstanceID, &r.CourseID, &r.EnrollmentID, &r.ToolID, &r.Role,
			&r.Engaged, &r.Completed, &r.ScorePct, &r.DurationMs, &facets, &r.ProjectionVersion, &r.UpdatedAt,
			&r.DisplayName, &r.Status,
		); err != nil {
			return nil, err
		}
		if len(facets) == 0 {
			facets = []byte(`{}`)
		}
		r.FacetsJSON = json.RawMessage(facets)
		out = append(out, r)
	}
	return out, rows.Err()
}
