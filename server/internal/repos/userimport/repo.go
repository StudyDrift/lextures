// Package userimport persists bulk user CSV import job state (plan 18.2).
package userimport

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/csvimport"
)

// Status mirrors provisioning.import_job_status.
type Status string

const (
	StatusQueued   Status = "queued"
	StatusRunning  Status = "running"
	StatusComplete Status = "complete"
	StatusFailed   Status = "failed"
)

// Job is a provisioning.user_import_jobs row.
type Job struct {
	ID               uuid.UUID
	OrgID            uuid.UUID
	ActorID          uuid.UUID
	Status           Status
	MergeStrategy    csvimport.MergeStrategy
	ImportProfile    csvimport.Profile
	DryRun           bool
	TotalRows        *int
	ProcessedRows    int
	ErrorRows        int
	CreatedCount     int
	UpdatedCount     int
	DeactivatedCount int
	SkippedCount     int
	ErrorsJSON       json.RawMessage
	CursorRow        int
	InputFilePath    *string
	ResultFilePath   *string
	QueueJobID       *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// RowError is re-exported for API responses.
type RowError = csvimport.RowError

// InsertParams creates a new import job.
type InsertParams struct {
	OrgID          uuid.UUID
	ActorID        uuid.UUID
	MergeStrategy  csvimport.MergeStrategy
	ImportProfile  csvimport.Profile
	DryRun         bool
	TotalRows      int
	InputFilePath  string
}

// Insert creates a queued import job.
func Insert(ctx context.Context, pool *pgxpool.Pool, p InsertParams) (*Job, error) {
	var j Job
	var strat, prof string
	err := pool.QueryRow(ctx, `
INSERT INTO provisioning.user_import_jobs
  (org_id, actor_id, merge_strategy, import_profile, dry_run, total_rows, input_file_path)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, actor_id, status, merge_strategy, import_profile, dry_run,
  total_rows, processed_rows, error_rows, created_count, updated_count, deactivated_count,
  skipped_count, errors_jsonb, cursor_row, input_file_path, result_file_path, queue_job_id,
  created_at, updated_at
`, p.OrgID, p.ActorID, string(p.MergeStrategy), string(p.ImportProfile), p.DryRun, p.TotalRows, p.InputFilePath).Scan(
		&j.ID, &j.OrgID, &j.ActorID, &j.Status, &strat, &prof, &j.DryRun,
		&j.TotalRows, &j.ProcessedRows, &j.ErrorRows, &j.CreatedCount, &j.UpdatedCount, &j.DeactivatedCount,
		&j.SkippedCount, &j.ErrorsJSON, &j.CursorRow, &j.InputFilePath, &j.ResultFilePath, &j.QueueJobID,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	j.MergeStrategy = csvimport.MergeStrategy(strat)
	j.ImportProfile = csvimport.Profile(prof)
	return &j, nil
}

// Get returns a job by id scoped to org, or nil.
func Get(ctx context.Context, pool *pgxpool.Pool, orgID, jobID uuid.UUID) (*Job, error) {
	return scanJob(ctx, pool, `
SELECT id, org_id, actor_id, status, merge_strategy, import_profile, dry_run,
  total_rows, processed_rows, error_rows, created_count, updated_count, deactivated_count,
  skipped_count, errors_jsonb, cursor_row, input_file_path, result_file_path, queue_job_id,
  created_at, updated_at
FROM provisioning.user_import_jobs
WHERE id = $1 AND org_id = $2
`, jobID, orgID)
}

// GetByID returns a job without org scope (worker use).
func GetByID(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID) (*Job, error) {
	return scanJob(ctx, pool, `
SELECT id, org_id, actor_id, status, merge_strategy, import_profile, dry_run,
  total_rows, processed_rows, error_rows, created_count, updated_count, deactivated_count,
  skipped_count, errors_jsonb, cursor_row, input_file_path, result_file_path, queue_job_id,
  created_at, updated_at
FROM provisioning.user_import_jobs
WHERE id = $1
`, jobID)
}

func scanJob(ctx context.Context, pool *pgxpool.Pool, q string, args ...any) (*Job, error) {
	var j Job
	var strat, prof string
	err := pool.QueryRow(ctx, q, args...).Scan(
		&j.ID, &j.OrgID, &j.ActorID, &j.Status, &strat, &prof, &j.DryRun,
		&j.TotalRows, &j.ProcessedRows, &j.ErrorRows, &j.CreatedCount, &j.UpdatedCount, &j.DeactivatedCount,
		&j.SkippedCount, &j.ErrorsJSON, &j.CursorRow, &j.InputFilePath, &j.ResultFilePath, &j.QueueJobID,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	j.MergeStrategy = csvimport.MergeStrategy(strat)
	j.ImportProfile = csvimport.Profile(prof)
	return &j, nil
}

// SetQueueJobID links the domain job to jobs.queue.
func SetQueueJobID(ctx context.Context, pool *pgxpool.Pool, jobID, queueID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE provisioning.user_import_jobs SET queue_job_id = $2, updated_at = NOW() WHERE id = $1
`, jobID, queueID)
	return err
}

// MarkRunning sets status to running.
func MarkRunning(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE provisioning.user_import_jobs SET status = 'running', updated_at = NOW() WHERE id = $1
`, jobID)
	return err
}

// ProgressUpdate persists cursor and counters during processing.
type ProgressUpdate struct {
	ProcessedRows    int
	ErrorRows        int
	CreatedCount     int
	UpdatedCount     int
	DeactivatedCount int
	SkippedCount     int
	CursorRow        int
	Errors           []RowError
}

// UpdateProgress saves in-flight progress (idempotent resume support).
func UpdateProgress(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, p ProgressUpdate) error {
	var errJSON []byte
	if len(p.Errors) > 0 {
		errJSON, _ = json.Marshal(p.Errors)
	}
	_, err := pool.Exec(ctx, `
UPDATE provisioning.user_import_jobs SET
  processed_rows = $2,
  error_rows = $3,
  created_count = $4,
  updated_count = $5,
  deactivated_count = $6,
  skipped_count = $7,
  cursor_row = $8,
  errors_jsonb = COALESCE($9::jsonb, errors_jsonb),
  updated_at = NOW()
WHERE id = $1
`, jobID, p.ProcessedRows, p.ErrorRows, p.CreatedCount, p.UpdatedCount, p.DeactivatedCount, p.SkippedCount, p.CursorRow, nullJSON(errJSON))
	return err
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// Complete marks a job finished and optionally stores result path; clears input path.
type CompleteParams struct {
	Status         Status
	ResultFilePath *string
	Errors         []RowError
	ProcessedRows  int
	ErrorRows      int
	CreatedCount   int
	UpdatedCount   int
	Deactivated    int
	Skipped        int
}

// Complete finalizes a job.
func Complete(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, p CompleteParams) error {
	var errJSON []byte
	if len(p.Errors) > 0 {
		errJSON, _ = json.Marshal(p.Errors)
	}
	_, err := pool.Exec(ctx, `
UPDATE provisioning.user_import_jobs SET
  status = $2,
  result_file_path = $3,
  input_file_path = NULL,
  processed_rows = $4,
  error_rows = $5,
  created_count = $6,
  updated_count = $7,
  deactivated_count = $8,
  skipped_count = $9,
  errors_jsonb = COALESCE($10::jsonb, errors_jsonb),
  updated_at = NOW()
WHERE id = $1
`, jobID, p.Status, p.ResultFilePath, p.ProcessedRows, p.ErrorRows, p.CreatedCount, p.UpdatedCount, p.Deactivated, p.Skipped, nullJSON(errJSON))
	return err
}

// ListRecent returns import jobs for an org.
func ListRecent(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, limit, offset int) ([]Job, int, error) {
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM provisioning.user_import_jobs WHERE org_id = $1`, orgID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := pool.Query(ctx, `
SELECT id, org_id, actor_id, status, merge_strategy, import_profile, dry_run,
  total_rows, processed_rows, error_rows, created_count, updated_count, deactivated_count,
  skipped_count, errors_jsonb, cursor_row, input_file_path, result_file_path, queue_job_id,
  created_at, updated_at
FROM provisioning.user_import_jobs
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]Job, 0)
	for rows.Next() {
		var j Job
		var strat, prof string
		if err := rows.Scan(
			&j.ID, &j.OrgID, &j.ActorID, &j.Status, &strat, &prof, &j.DryRun,
			&j.TotalRows, &j.ProcessedRows, &j.ErrorRows, &j.CreatedCount, &j.UpdatedCount, &j.DeactivatedCount,
			&j.SkippedCount, &j.ErrorsJSON, &j.CursorRow, &j.InputFilePath, &j.ResultFilePath, &j.QueueJobID,
			&j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		j.MergeStrategy = csvimport.MergeStrategy(strat)
		j.ImportProfile = csvimport.Profile(prof)
		out = append(out, j)
	}
	return out, total, rows.Err()
}

// CountRecentUploads counts jobs created in the last hour for rate limiting.
func CountRecentUploads(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM provisioning.user_import_jobs
WHERE org_id = $1 AND created_at >= $2
`, orgID, since).Scan(&n)
	return n, err
}
