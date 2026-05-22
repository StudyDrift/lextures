// Package reportschedules manages recurring report delivery schedules (plan 9.8).
package reportschedules

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Schedule is a stored report delivery configuration.
type Schedule struct {
	ID           uuid.UUID
	OwnerID      uuid.UUID
	CourseID     *uuid.UUID
	ReportType   string
	Parameters   map[string]string
	Recipients   []string
	Cadence      string
	CadenceDetail map[string]any
	Enabled      bool
	LastRunAt    *time.Time
	NextRunAt    time.Time
	CreatedAt    time.Time
}

// Create inserts a new schedule and returns it with the generated ID.
func Create(ctx context.Context, pool *pgxpool.Pool, s Schedule) (Schedule, error) {
	params, _ := json.Marshal(s.Parameters)
	detail, _ := json.Marshal(s.CadenceDetail)
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO analytics.report_schedules
  (owner_id, course_id, report_type, parameters, recipients, cadence, cadence_detail, next_run_at)
VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7::jsonb, $8)
RETURNING id
`, s.OwnerID, s.CourseID, s.ReportType, params, s.Recipients, s.Cadence, detail, s.NextRunAt).Scan(&id)
	if err != nil {
		return Schedule{}, err
	}
	s.ID = id
	return s, nil
}

// List returns all schedules owned by a user.
func List(ctx context.Context, pool *pgxpool.Pool, ownerID uuid.UUID) ([]Schedule, error) {
	rows, err := pool.Query(ctx, `
SELECT id, owner_id, course_id, report_type, parameters, recipients, cadence, cadence_detail,
       enabled, last_run_at, next_run_at, created_at
FROM analytics.report_schedules
WHERE owner_id = $1
ORDER BY created_at DESC
`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// Get returns a single schedule by ID.
func Get(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Schedule, error) {
	rows, err := pool.Query(ctx, `
SELECT id, owner_id, course_id, report_type, parameters, recipients, cadence, cadence_detail,
       enabled, last_run_at, next_run_at, created_at
FROM analytics.report_schedules
WHERE id = $1
`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

// UpdateInput holds the fields that may be changed on an existing schedule.
type UpdateInput struct {
	Recipients    []string
	Cadence       string
	CadenceDetail map[string]any
	Enabled       bool
	NextRunAt     time.Time
}

// Update replaces mutable fields on an existing schedule.
func Update(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, in UpdateInput) error {
	detail, _ := json.Marshal(in.CadenceDetail)
	_, err := pool.Exec(ctx, `
UPDATE analytics.report_schedules
SET recipients = $2, cadence = $3, cadence_detail = $4::jsonb,
    enabled = $5, next_run_at = $6
WHERE id = $1
`, id, in.Recipients, in.Cadence, detail, in.Enabled, in.NextRunAt)
	return err
}

// Delete removes a schedule permanently.
func Delete(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM analytics.report_schedules WHERE id = $1`, id)
	return err
}

// ListDue returns enabled schedules whose next_run_at is at or before now.
func ListDue(ctx context.Context, pool *pgxpool.Pool, now time.Time, limit int) ([]Schedule, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := pool.Query(ctx, `
SELECT id, owner_id, course_id, report_type, parameters, recipients, cadence, cadence_detail,
       enabled, last_run_at, next_run_at, created_at
FROM analytics.report_schedules
WHERE enabled = true AND next_run_at <= $1
ORDER BY next_run_at
LIMIT $2
`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// MarkRan records a successful run and advances next_run_at.
func MarkRan(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, ranAt, nextRun time.Time) error {
	_, err := pool.Exec(ctx, `
UPDATE analytics.report_schedules
SET last_run_at = $2, next_run_at = $3
WHERE id = $1
`, id, ranAt, nextRun)
	return err
}

func scanRows(rows pgx.Rows) ([]Schedule, error) {
	var out []Schedule
	for rows.Next() {
		var s Schedule
		var paramsRaw, detailRaw []byte
		if err := rows.Scan(
			&s.ID, &s.OwnerID, &s.CourseID, &s.ReportType, &paramsRaw, &s.Recipients,
			&s.Cadence, &detailRaw, &s.Enabled, &s.LastRunAt, &s.NextRunAt, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		s.Parameters = map[string]string{}
		if len(paramsRaw) > 0 {
			_ = json.Unmarshal(paramsRaw, &s.Parameters)
		}
		s.CadenceDetail = map[string]any{}
		if len(detailRaw) > 0 {
			_ = json.Unmarshal(detailRaw, &s.CadenceDetail)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
