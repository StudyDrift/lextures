package lrsforwardjobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Job is a queued LRS forward attempt.
type Job struct {
	ID                uuid.UUID
	StatementID       uuid.UUID
	StatementStoredAt time.Time
	LRSEndpointID     uuid.UUID
	Status            string
	Attempts          int
	NextRetryAt       *time.Time
	LastHTTPStatus    *int
	LastResponse      *string
}

// EnqueueForEndpoints creates pending jobs for each endpoint id.
func EnqueueForEndpoints(ctx context.Context, pool *pgxpool.Pool, statementID uuid.UUID, storedAt time.Time, endpointIDs []uuid.UUID) error {
	for _, eid := range endpointIDs {
		_, err := pool.Exec(ctx, `
INSERT INTO analytics.lrs_forward_jobs (statement_id, statement_stored_at, lrs_endpoint_id)
VALUES ($1, $2, $3)
`, statementID, storedAt.UTC(), eid)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListDue returns jobs ready to process.
func ListDue(ctx context.Context, pool *pgxpool.Pool, limit int, now time.Time) ([]Job, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT id, statement_id, statement_stored_at, lrs_endpoint_id, status, attempts, next_retry_at, last_http_status, last_response
FROM analytics.lrs_forward_jobs
WHERE status IN ('pending', 'failed')
  AND (next_retry_at IS NULL OR next_retry_at <= $1)
ORDER BY created_at
LIMIT $2
`, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.StatementID, &j.StatementStoredAt, &j.LRSEndpointID,
			&j.Status, &j.Attempts, &j.NextRetryAt, &j.LastHTTPStatus, &j.LastResponse); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// MarkSent marks a job delivered.
func MarkSent(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, at time.Time, httpStatus int, responseSnippet string) error {
	_, err := pool.Exec(ctx, `
UPDATE analytics.lrs_forward_jobs
SET status = 'sent', sent_at = $2, last_http_status = $3, last_response = $4, next_retry_at = NULL
WHERE id = $1
`, jobID, at.UTC(), httpStatus, truncate(responseSnippet, 2000))
	return err
}

// MarkRetry schedules retry or dead-letter.
func MarkRetry(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, attempts int, nextRetry time.Time, dead bool, httpStatus int, errMsg string) error {
	status := "failed"
	if dead {
		status = "dead"
	}
	_, err := pool.Exec(ctx, `
UPDATE analytics.lrs_forward_jobs
SET status = $2, attempts = $3, next_retry_at = $4, last_http_status = $5, last_response = $6
WHERE id = $1
`, jobID, status, attempts, nullableTime(nextRetry, dead), httpStatus, truncate(errMsg, 2000))
	return err
}

// InsertDeadLetter records a permanent forwarding failure.
func InsertDeadLetter(ctx context.Context, pool *pgxpool.Pool, statementID uuid.UUID, storedAt time.Time, endpointID uuid.UUID, lastError string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.lrs_dead_letter (statement_id, statement_stored_at, lrs_endpoint_id, last_error)
VALUES ($1, $2, $3, $4)
`, statementID, storedAt.UTC(), endpointID, truncate(lastError, 4000))
	return err
}

// ListDeadLetter returns recent dead-letter rows for admin UI.
func ListDeadLetter(ctx context.Context, pool *pgxpool.Pool, limit int) ([]DeadLetterRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, statement_id, statement_stored_at, lrs_endpoint_id, last_error, created_at
FROM analytics.lrs_dead_letter
ORDER BY created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeadLetterRow
	for rows.Next() {
		var r DeadLetterRow
		if err := rows.Scan(&r.ID, &r.StatementID, &r.StatementStoredAt, &r.LRSEndpointID, &r.LastError, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RetryDeadLetter re-queues a dead-letter statement for one endpoint.
func RetryDeadLetter(ctx context.Context, pool *pgxpool.Pool, deadLetterID uuid.UUID) (bool, error) {
	var stmtID uuid.UUID
	var storedAt time.Time
	var endpointID uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT statement_id, statement_stored_at, lrs_endpoint_id
FROM analytics.lrs_dead_letter WHERE id = $1
`, deadLetterID).Scan(&stmtID, &storedAt, &endpointID)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO analytics.lrs_forward_jobs (statement_id, statement_stored_at, lrs_endpoint_id, status)
VALUES ($1, $2, $3, 'pending')
`, stmtID, storedAt.UTC(), endpointID)
	if err != nil {
		return false, err
	}
	_, _ = pool.Exec(ctx, `DELETE FROM analytics.lrs_dead_letter WHERE id = $1`, deadLetterID)
	return true, nil
}

// DeadLetterRow is an admin list item.
type DeadLetterRow struct {
	ID                uuid.UUID
	StatementID       uuid.UUID
	StatementStoredAt time.Time
	LRSEndpointID     uuid.UUID
	LastError         *string
	CreatedAt         time.Time
}

func nullableTime(t time.Time, dead bool) *time.Time {
	if dead {
		return nil
	}
	u := t.UTC()
	return &u
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
