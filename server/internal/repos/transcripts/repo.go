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

// DeliveryType is how a student wants their transcript delivered.
type DeliveryType string

const (
	DeliveryEmail  DeliveryType = "email"
	DeliveryMail   DeliveryType = "mail"
	DeliveryPickup DeliveryType = "pickup"
)

// UrgencyUnit distinguishes calendar days from business days.
type UrgencyUnit string

const (
	UrgencyDays         UrgencyUnit = "days"
	UrgencyBusinessDays UrgencyUnit = "business_days"
)

// Request is a student transcript request row.
type Request struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	OrgID               *uuid.UUID
	Status              RequestStatus
	DeliveryType        DeliveryType
	DeliveryEmail       *string
	DeliveryAddress     *string
	UrgencyDays         int
	UrgencyDaysMin      *int
	UrgencyUnit         UrgencyUnit
	ErrorMessage        *string
	WebhookResponseCode *int
	RequestedAt         time.Time
	SubmittedAt         *time.Time
	CreatedAt           time.Time
}

// InsertRequestInput captures delivery preferences for a new transcript request.
type InsertRequestInput struct {
	DeliveryType    DeliveryType
	DeliveryEmail   *string
	DeliveryAddress *string
	UrgencyDays     int
	UrgencyDaysMin  *int
	UrgencyUnit     UrgencyUnit
}

// Config holds the institution webhook settings.
type Config struct {
	WebhookURL          *string
	WebhookSecret       *string
	PickupInstructions  *string
	UpdatedAt           time.Time
}

// GetConfig returns the singleton transcripts config row.
func GetConfig(ctx context.Context, pool *pgxpool.Pool) (*Config, error) {
	var c Config
	var url, secret, pickup *string
	err := pool.QueryRow(ctx, `
SELECT webhook_url, webhook_secret, pickup_instructions, updated_at
FROM settings.transcripts_config
WHERE id = 1
`).Scan(&url, &secret, &pickup, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	c.WebhookURL = url
	c.WebhookSecret = secret
	c.PickupInstructions = pickup
	return &c, nil
}

// UpsertConfig saves webhook URL, optional secret (empty secret leaves unchanged), and pickup instructions.
func UpsertConfig(
	ctx context.Context,
	pool *pgxpool.Pool,
	webhookURL string,
	webhookSecret *string,
	pickupInstructions *string,
) (*Config, error) {
	var c Config
	var url, secret, pickup *string
	err := pool.QueryRow(ctx, `
UPDATE settings.transcripts_config
SET
    webhook_url = $1,
    webhook_secret = CASE
        WHEN $2::text IS NOT NULL AND TRIM($2) <> '' THEN TRIM($2)
        ELSE webhook_secret
    END,
    pickup_instructions = CASE
        WHEN $3::text IS NOT NULL THEN NULLIF(TRIM($3), '')
        ELSE pickup_instructions
    END,
    updated_at = NOW()
WHERE id = 1
RETURNING webhook_url, webhook_secret, pickup_instructions, updated_at
`, webhookURL, webhookSecret, pickupInstructions).Scan(&url, &secret, &pickup, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.WebhookURL = url
	c.WebhookSecret = secret
	c.PickupInstructions = pickup
	return &c, nil
}

// requestSelectColumns is the shared SELECT list for transcript request queries.
const requestSelectColumns = `
id, user_id, org_id, status, delivery_type, delivery_email, delivery_address,
urgency_days, urgency_days_min, urgency_unit, error_message, webhook_response_code,
requested_at, submitted_at, created_at`

func scanRequestRow(row pgx.Row, r *Request) error {
	var orgIDScan *uuid.UUID
	var deliveryType string
	var urgencyUnit string
	err := row.Scan(
		&r.ID, &r.UserID, &orgIDScan, &r.Status, &deliveryType, &r.DeliveryEmail, &r.DeliveryAddress,
		&r.UrgencyDays, &r.UrgencyDaysMin, &urgencyUnit, &r.ErrorMessage,
		&r.WebhookResponseCode, &r.RequestedAt, &r.SubmittedAt, &r.CreatedAt,
	)
	if err != nil {
		return err
	}
	r.DeliveryType = DeliveryType(deliveryType)
	r.UrgencyUnit = UrgencyUnit(urgencyUnit)
	r.OrgID = orgIDScan
	return nil
}

// InsertRequest creates a new queued transcript request.
func InsertRequest(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	orgID *uuid.UUID,
	input InsertRequestInput,
) (*Request, error) {
	var r Request
	row := pool.QueryRow(ctx, `
INSERT INTO transcripts.transcript_requests (
    user_id, org_id, delivery_type, delivery_email, delivery_address,
    urgency_days, urgency_days_min, urgency_unit
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING `+requestSelectColumns+`
`, userID, orgID, input.DeliveryType, input.DeliveryEmail, input.DeliveryAddress,
		input.UrgencyDays, input.UrgencyDaysMin, input.UrgencyUnit)
	if err := scanRequestRow(row, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListByUser returns transcript requests for a user, newest first.
func ListByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Request, error) {
	rows, err := pool.Query(ctx, `
SELECT `+requestSelectColumns+`
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
		if err := scanRequestRow(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListFailed returns failed transcript requests for an org, newest first.
func ListFailed(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Request, error) {
	rows, err := pool.Query(ctx, `
SELECT `+requestSelectColumns+`
FROM transcripts.transcript_requests
WHERE org_id = $1 AND status = 'failed'
ORDER BY requested_at DESC
LIMIT 100
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request
		if err := scanRequestRow(rows, &r); err != nil {
			return nil, err
		}
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
