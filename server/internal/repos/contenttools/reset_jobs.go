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

// ResetJobRow is a course.content_tool_reset_jobs row.
type ResetJobRow struct {
	ID             uuid.UUID
	CourseID       uuid.UUID
	RequestedBy    *uuid.UUID
	Scope          string
	TargetJSON     json.RawMessage
	Reason         *string
	Notify         bool
	Status         string
	TotalRows      int
	ProcessedRows  int
	BatchID        *uuid.UUID
	Error          *string
	ResultJSON     json.RawMessage
	IdempotencyKey *string
	CreatedAt      time.Time
	FinishedAt     *time.Time
}

// InsertResetJob creates a queued async reset job. On unique idempotency conflict returns existing.
func InsertResetJob(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	requestedBy uuid.UUID,
	scope string,
	target json.RawMessage,
	reason *string,
	notify bool,
	idempotencyKey *string,
	totalRows int,
) (*ResetJobRow, bool, error) {
	if len(target) == 0 {
		target = json.RawMessage(`{}`)
	}
	if idempotencyKey != nil && *idempotencyKey != "" {
		existing, err := GetResetJobByIdempotency(ctx, pool, *idempotencyKey)
		if err != nil {
			return nil, false, err
		}
		if existing != nil {
			return existing, true, nil
		}
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_reset_jobs (
  course_id, requested_by, scope, target_json, reason, notify, status, total_rows, idempotency_key
) VALUES ($1,$2,$3,$4::jsonb,$5,$6,'queued',$7,$8)
RETURNING id, course_id, requested_by, scope, target_json, reason, notify, status,
  total_rows, processed_rows, batch_id, error, result_json, idempotency_key, created_at, finished_at
`, courseID, requestedBy, scope, []byte(target), reason, notify, totalRows, idempotencyKey)
	job, err := scanResetJob(row)
	if err != nil {
		// Race on unique key — return existing.
		if idempotencyKey != nil && *idempotencyKey != "" {
			existing, e2 := GetResetJobByIdempotency(ctx, pool, *idempotencyKey)
			if e2 == nil && existing != nil {
				return existing, true, nil
			}
		}
		return nil, false, err
	}
	return job, false, nil
}

func scanResetJob(row pgx.Row) (*ResetJobRow, error) {
	var r ResetJobRow
	var target, result []byte
	err := row.Scan(
		&r.ID, &r.CourseID, &r.RequestedBy, &r.Scope, &target, &r.Reason, &r.Notify, &r.Status,
		&r.TotalRows, &r.ProcessedRows, &r.BatchID, &r.Error, &result, &r.IdempotencyKey, &r.CreatedAt, &r.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(target) == 0 {
		target = []byte(`{}`)
	}
	r.TargetJSON = json.RawMessage(target)
	if len(result) > 0 {
		r.ResultJSON = json.RawMessage(result)
	}
	return &r, nil
}

// GetResetJob returns a job by id.
func GetResetJob(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID) (*ResetJobRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, course_id, requested_by, scope, target_json, reason, notify, status,
  total_rows, processed_rows, batch_id, error, result_json, idempotency_key, created_at, finished_at
FROM course.content_tool_reset_jobs
WHERE id = $1
`, jobID)
	return scanResetJob(row)
}

// GetResetJobByIdempotency returns a job by idempotency key.
func GetResetJobByIdempotency(ctx context.Context, pool *pgxpool.Pool, key string) (*ResetJobRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, course_id, requested_by, scope, target_json, reason, notify, status,
  total_rows, processed_rows, batch_id, error, result_json, idempotency_key, created_at, finished_at
FROM course.content_tool_reset_jobs
WHERE idempotency_key = $1
`, key)
	return scanResetJob(row)
}

// MarkResetJobRunning sets status to running.
func MarkResetJobRunning(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE course.content_tool_reset_jobs
SET status = 'running'
WHERE id = $1 AND status = 'queued'
`, jobID)
	return err
}

// UpdateResetJobProgress updates processed row count.
func UpdateResetJobProgress(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, processed int) error {
	_, err := pool.Exec(ctx, `
UPDATE course.content_tool_reset_jobs
SET processed_rows = $2
WHERE id = $1
`, jobID, processed)
	return err
}

// FinishResetJob marks the job succeeded or failed.
func FinishResetJob(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, status string, batchID *uuid.UUID, result json.RawMessage, errMsg *string) error {
	if len(result) == 0 {
		result = json.RawMessage(`{}`)
	}
	_, err := pool.Exec(ctx, `
UPDATE course.content_tool_reset_jobs
SET status = $2,
    batch_id = $3,
    result_json = $4::jsonb,
    error = $5,
    finished_at = NOW(),
    processed_rows = CASE WHEN $2 = 'succeeded' THEN total_rows ELSE processed_rows END
WHERE id = $1
`, jobID, status, batchID, []byte(result), errMsg)
	return err
}
