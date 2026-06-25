package payments

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

// WebhookJob is a queued provider webhook payload.
type WebhookJob struct {
	ID              uuid.UUID
	Provider        string
	ProviderEventID string
	Payload         []byte
	Headers         map[string]string
	Status          string
	Attempts        int
	NextRetryAt     *time.Time
	LastError       *string
	CreatedAt       time.Time
	ProcessedAt     *time.Time
}

// EnqueueWebhook inserts a webhook job idempotently by provider event id.
func EnqueueWebhook(ctx context.Context, pool *pgxpool.Pool, provider, eventID string, payload []byte, headers map[string]string) (uuid.UUID, bool, error) {
	if eventID == "" {
		return uuid.Nil, false, errors.New("provider_event_id required")
	}
	if headers == nil {
		headers = map[string]string{}
	}
	rawHeaders, err := json.Marshal(headers)
	if err != nil {
		return uuid.Nil, false, err
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO payments.webhook_jobs (provider, provider_event_id, payload, headers)
VALUES ($1, $2, $3::jsonb, $4::jsonb)
ON CONFLICT (provider_event_id) DO NOTHING
RETURNING id
`, provider, eventID, payload, rawHeaders).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, err
	}
	err = pool.QueryRow(ctx, `SELECT id FROM payments.webhook_jobs WHERE provider_event_id = $1`, eventID).Scan(&id)
	return id, false, err
}

// ListDueWebhookJobs returns jobs ready for processing.
func ListDueWebhookJobs(ctx context.Context, pool *pgxpool.Pool, limit int, now time.Time) ([]WebhookJob, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT id, provider, provider_event_id, payload, headers, status, attempts, next_retry_at, last_error, created_at, processed_at
FROM payments.webhook_jobs
WHERE status IN ('pending', 'failed')
  AND (next_retry_at IS NULL OR next_retry_at <= $1)
ORDER BY created_at
LIMIT $2
FOR UPDATE SKIP LOCKED
`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebhookJob
	for rows.Next() {
		var j WebhookJob
		var rawPayload, rawHeaders []byte
		if err := rows.Scan(
			&j.ID, &j.Provider, &j.ProviderEventID, &rawPayload, &rawHeaders,
			&j.Status, &j.Attempts, &j.NextRetryAt, &j.LastError, &j.CreatedAt, &j.ProcessedAt,
		); err != nil {
			return nil, err
		}
		j.Payload = rawPayload
		j.Headers = map[string]string{}
		if len(rawHeaders) > 0 {
			_ = json.Unmarshal(rawHeaders, &j.Headers)
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// MarkWebhookProcessing locks a job for processing.
func MarkWebhookProcessing(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE payments.webhook_jobs SET status = 'processing' WHERE id = $1 AND status IN ('pending', 'failed')
`, id)
	return err
}

// MarkWebhookCompleted marks a job done.
func MarkWebhookCompleted(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, now time.Time) error {
	_, err := pool.Exec(ctx, `
UPDATE payments.webhook_jobs SET status = 'completed', processed_at = $2 WHERE id = $1
`, id, now)
	return err
}

// MarkWebhookFailed schedules retry or marks dead.
func MarkWebhookFailed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, attempts int, nextRetry *time.Time, errMsg string, dead bool) error {
	status := JobStatusFailed
	if dead {
		status = JobStatusFailed
	}
	_, err := pool.Exec(ctx, `
UPDATE payments.webhook_jobs
SET status = $2, attempts = $3, next_retry_at = $4, last_error = $5
WHERE id = $1
`, id, status, attempts, nextRetry, errMsg)
	return err
}
