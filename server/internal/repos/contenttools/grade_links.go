package contenttools

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GradeLinkRow is course.content_tool_grade_links.
type GradeLinkRow struct {
	InstanceID       uuid.UUID
	AssignmentItemID *uuid.UUID
	OutcomeID        *uuid.UUID
	PointsPossible   *float64
	CountsForGrade   bool
	LatePolicy       string
	EnabledBy        *uuid.UUID
	EnabledAt        time.Time
}

// GetGradeLink returns the grade link for an instance, or nil.
func GetGradeLink(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID) (*GradeLinkRow, error) {
	row := pool.QueryRow(ctx, `
SELECT instance_id, assignment_item_id, outcome_id, points_possible, counts_for_grade, late_policy, enabled_by, enabled_at
FROM course.content_tool_grade_links
WHERE instance_id = $1
`, instanceID)
	return scanGradeLink(row)
}

// UpsertGradeLink creates or updates a grade link (opt-in bridge).
func UpsertGradeLink(ctx context.Context, pool *pgxpool.Pool, link GradeLinkRow) (*GradeLinkRow, error) {
	if link.LatePolicy == "" {
		link.LatePolicy = "accept"
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_grade_links (
  instance_id, assignment_item_id, outcome_id, points_possible, counts_for_grade, late_policy, enabled_by, enabled_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
ON CONFLICT (instance_id) DO UPDATE SET
  assignment_item_id = EXCLUDED.assignment_item_id,
  outcome_id = EXCLUDED.outcome_id,
  points_possible = EXCLUDED.points_possible,
  counts_for_grade = EXCLUDED.counts_for_grade,
  late_policy = EXCLUDED.late_policy,
  enabled_by = EXCLUDED.enabled_by,
  enabled_at = NOW()
RETURNING instance_id, assignment_item_id, outcome_id, points_possible, counts_for_grade, late_policy, enabled_by, enabled_at
`, link.InstanceID, link.AssignmentItemID, link.OutcomeID, link.PointsPossible, link.CountsForGrade, link.LatePolicy, link.EnabledBy)
	return scanGradeLink(row)
}

// DeleteGradeLink removes the opt-in bridge for an instance.
func DeleteGradeLink(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM course.content_tool_grade_links WHERE instance_id = $1`, instanceID)
	return err
}

// ListGradeLinksForInstances returns grade links keyed by instance id.
func ListGradeLinksForInstances(ctx context.Context, pool *pgxpool.Pool, instanceIDs []uuid.UUID) (map[uuid.UUID]GradeLinkRow, error) {
	out := map[uuid.UUID]GradeLinkRow{}
	if len(instanceIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
SELECT instance_id, assignment_item_id, outcome_id, points_possible, counts_for_grade, late_policy, enabled_by, enabled_at
FROM course.content_tool_grade_links
WHERE instance_id = ANY($1::uuid[])
`, instanceIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanGradeLink(rows)
		if err != nil {
			return nil, err
		}
		if r != nil {
			out[r.InstanceID] = *r
		}
	}
	return out, rows.Err()
}

func scanGradeLink(row pgx.Row) (*GradeLinkRow, error) {
	var r GradeLinkRow
	err := row.Scan(
		&r.InstanceID, &r.AssignmentItemID, &r.OutcomeID, &r.PointsPossible,
		&r.CountsForGrade, &r.LatePolicy, &r.EnabledBy, &r.EnabledAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if r.LatePolicy == "" {
		r.LatePolicy = "accept"
	}
	return &r, nil
}
