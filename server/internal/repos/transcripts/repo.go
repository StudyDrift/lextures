package transcripts

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrWebhookNotConfigured = errors.New("transcript webhook not configured")

// RequestStatus is the lifecycle state of a transcript request.
type RequestStatus string

const (
	StatusQueued    RequestStatus = "queued"
	StatusSubmitted RequestStatus = "submitted"
	StatusFailed    RequestStatus = "failed"
)

// Request is a student transcript request row.
type Request struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	OrgID               *uuid.UUID
	Status              RequestStatus
	ErrorMessage        *string
	WebhookResponseCode *int
	RequestedAt         time.Time
	SubmittedAt         *time.Time
	CreatedAt           time.Time
}

// Config holds the institution webhook settings.
type Config struct {
	WebhookURL    *string
	WebhookSecret *string
	UpdatedAt     time.Time
}

// GetConfig returns the singleton transcripts config row.
func GetConfig(ctx context.Context, pool *pgxpool.Pool) (*Config, error) {
	var c Config
	var url, secret *string
	err := pool.QueryRow(ctx, `
SELECT webhook_url, webhook_secret, updated_at
FROM settings.transcripts_config
WHERE id = 1
`).Scan(&url, &secret, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	c.WebhookURL = url
	c.WebhookSecret = secret
	return &c, nil
}

// UpsertConfig saves webhook URL and optional secret (empty secret leaves unchanged).
func UpsertConfig(ctx context.Context, pool *pgxpool.Pool, webhookURL string, webhookSecret *string) (*Config, error) {
	var c Config
	var url, secret *string
	err := pool.QueryRow(ctx, `
UPDATE settings.transcripts_config
SET
    webhook_url = $1,
    webhook_secret = CASE
        WHEN $2::text IS NOT NULL AND TRIM($2) <> '' THEN TRIM($2)
        ELSE webhook_secret
    END,
    updated_at = NOW()
WHERE id = 1
RETURNING webhook_url, webhook_secret, updated_at
`, webhookURL, webhookSecret).Scan(&url, &secret, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.WebhookURL = url
	c.WebhookSecret = secret
	return &c, nil
}

// InsertRequest creates a new queued transcript request.
func InsertRequest(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, orgID *uuid.UUID) (*Request, error) {
	var r Request
	var orgIDScan *uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO transcripts.transcript_requests (user_id, org_id)
VALUES ($1, $2)
RETURNING id, user_id, org_id, status, error_message, webhook_response_code,
          requested_at, submitted_at, created_at
`, userID, orgID).Scan(
		&r.ID, &r.UserID, &orgIDScan, &r.Status, &r.ErrorMessage,
		&r.WebhookResponseCode, &r.RequestedAt, &r.SubmittedAt, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.OrgID = orgIDScan
	return &r, nil
}

// ListByUser returns transcript requests for a user, newest first.
func ListByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Request, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, org_id, status, error_message, webhook_response_code,
       requested_at, submitted_at, created_at
FROM transcripts.transcript_requests
WHERE user_id = $1
ORDER BY requested_at DESC
LIMIT 50
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request
		var orgIDScan *uuid.UUID
		if err := rows.Scan(
			&r.ID, &r.UserID, &orgIDScan, &r.Status, &r.ErrorMessage,
			&r.WebhookResponseCode, &r.RequestedAt, &r.SubmittedAt, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		r.OrgID = orgIDScan
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkSubmitted updates a request after successful webhook delivery.
func MarkSubmitted(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, responseCode int) error {
	_, err := pool.Exec(ctx, `
UPDATE transcripts.transcript_requests
SET status = 'submitted', webhook_response_code = $2, submitted_at = NOW()
WHERE id = $1
`, id, responseCode)
	return err
}

// MarkFailed updates a request after webhook delivery failure.
func MarkFailed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, message string, responseCode *int) error {
	_, err := pool.Exec(ctx, `
UPDATE transcripts.transcript_requests
SET status = 'failed', error_message = $2, webhook_response_code = $3
WHERE id = $1
`, id, message, responseCode)
	return err
}
