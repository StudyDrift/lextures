package webhooksrepo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Subscription is an org webhook endpoint registration.
type Subscription struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	Label         string
	EndpointURL   string
	SigningKeyEnc string
	EventTypes    []string
	Active        bool
	PausedAt      *time.Time
	TLSSkipVerify bool
	CreatedBy     *uuid.UUID
	Settings      json.RawMessage
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Delivery is one webhook delivery attempt record.
type Delivery struct {
	ID             int64
	SubscriptionID uuid.UUID
	EventType      string
	EventID        uuid.UUID
	PayloadHash    string
	PayloadJSON    []byte
	AttemptCount   int
	Status         string
	LastHTTPStatus *int
	LastResponse   *string
	LatencyMS      *int
	NextRetryAt    *time.Time
	DeliveredAt    *time.Time
	CreatedAt      time.Time
}

// CreateInput holds fields for a new subscription.
type CreateInput struct {
	OrgID         uuid.UUID
	Label         string
	EndpointURL   string
	SigningKeyEnc string
	EventTypes    []string
	TLSSkipVerify bool
	CreatedBy     *uuid.UUID
	Settings      json.RawMessage
}

// UpdateInput holds mutable subscription fields.
type UpdateInput struct {
	Label         *string
	EndpointURL   *string
	SigningKeyEnc *string
	EventTypes    []string
	Active        *bool
	TLSSkipVerify *bool
	Reactivate    bool
}

// ListByOrg returns subscriptions for an organization.
func ListByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Subscription, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, label, endpoint_url, signing_key_enc, event_types, active, paused_at,
       tls_skip_verify, created_by, settings, created_at, updated_at
FROM integrations.webhook_subscriptions
WHERE org_id = $1
ORDER BY created_at DESC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.OrgID, &s.Label, &s.EndpointURL, &s.SigningKeyEnc, &s.EventTypes,
			&s.Active, &s.PausedAt, &s.TLSSkipVerify, &s.CreatedBy, &s.Settings, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetByID returns a subscription scoped to org.
func GetByID(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) (*Subscription, error) {
	row := pool.QueryRow(ctx, `
SELECT id, org_id, label, endpoint_url, signing_key_enc, event_types, active, paused_at,
       tls_skip_verify, created_by, settings, created_at, updated_at
FROM integrations.webhook_subscriptions
WHERE id = $1 AND org_id = $2
`, id, orgID)
	return scanSubscription(row)
}

// GetByIDAnyOrg loads a subscription by id (delivery worker).
func GetByIDAnyOrg(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Subscription, error) {
	row := pool.QueryRow(ctx, `
SELECT id, org_id, label, endpoint_url, signing_key_enc, event_types, active, paused_at,
       tls_skip_verify, created_by, settings, created_at, updated_at
FROM integrations.webhook_subscriptions
WHERE id = $1
`, id)
	return scanSubscription(row)
}

func scanSubscription(row pgx.Row) (*Subscription, error) {
	var s Subscription
	err := row.Scan(&s.ID, &s.OrgID, &s.Label, &s.EndpointURL, &s.SigningKeyEnc, &s.EventTypes,
		&s.Active, &s.PausedAt, &s.TLSSkipVerify, &s.CreatedBy, &s.Settings, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Create inserts a subscription.
func Create(ctx context.Context, pool *pgxpool.Pool, in CreateInput) (*Subscription, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO integrations.webhook_subscriptions (
    org_id, label, endpoint_url, signing_key_enc, event_types, tls_skip_verify, created_by, settings
) VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, '{}'::jsonb))
RETURNING id, org_id, label, endpoint_url, signing_key_enc, event_types, active, paused_at,
          tls_skip_verify, created_by, settings, created_at, updated_at
`, in.OrgID, in.Label, in.EndpointURL, in.SigningKeyEnc, in.EventTypes, in.TLSSkipVerify, in.CreatedBy, in.Settings)
	return scanSubscription(row)
}

// Update modifies a subscription.
func Update(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID, in UpdateInput) (*Subscription, error) {
	cur, err := GetByID(ctx, pool, orgID, id)
	if err != nil || cur == nil {
		return cur, err
	}
	label := cur.Label
	endpoint := cur.EndpointURL
	keyEnc := cur.SigningKeyEnc
	eventTypes := cur.EventTypes
	active := cur.Active
	tlsSkip := cur.TLSSkipVerify
	if in.Label != nil {
		label = *in.Label
	}
	if in.EndpointURL != nil {
		endpoint = *in.EndpointURL
	}
	if in.SigningKeyEnc != nil {
		keyEnc = *in.SigningKeyEnc
	}
	if in.EventTypes != nil {
		eventTypes = in.EventTypes
	}
	if in.Active != nil {
		active = *in.Active
	}
	if in.TLSSkipVerify != nil {
		tlsSkip = *in.TLSSkipVerify
	}
	var pausedAt *time.Time
	if in.Reactivate || (in.Active != nil && *in.Active) {
		pausedAt = nil
		active = true
	} else if !active {
		now := time.Now().UTC()
		pausedAt = &now
	} else {
		pausedAt = cur.PausedAt
	}
	row := pool.QueryRow(ctx, `
UPDATE integrations.webhook_subscriptions
SET label = $3, endpoint_url = $4, signing_key_enc = $5, event_types = $6,
    active = $7, paused_at = $8, tls_skip_verify = $9, updated_at = now()
WHERE id = $1 AND org_id = $2
RETURNING id, org_id, label, endpoint_url, signing_key_enc, event_types, active, paused_at,
          tls_skip_verify, created_by, settings, created_at, updated_at
`, id, orgID, label, endpoint, keyEnc, eventTypes, active, pausedAt, tlsSkip)
	return scanSubscription(row)
}

// Delete removes a subscription.
func Delete(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM integrations.webhook_subscriptions WHERE id = $1 AND org_id = $2
`, id, orgID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// Pause marks a subscription inactive after delivery failures.
func Pause(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE integrations.webhook_subscriptions
SET active = false, paused_at = now(), updated_at = now()
WHERE id = $1
`, id)
	return err
}

// ListActiveForEvent returns active subscriptions for org + event type.
func ListActiveForEvent(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, eventType string) ([]Subscription, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, label, endpoint_url, signing_key_enc, event_types, active, paused_at,
       tls_skip_verify, created_by, settings, created_at, updated_at
FROM integrations.webhook_subscriptions
WHERE org_id = $1 AND active = true AND paused_at IS NULL AND $2 = ANY(event_types)
`, orgID, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.OrgID, &s.Label, &s.EndpointURL, &s.SigningKeyEnc, &s.EventTypes,
			&s.Active, &s.PausedAt, &s.TLSSkipVerify, &s.CreatedBy, &s.Settings, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// InsertDelivery creates a pending delivery row with payload stored in last_response until sent.
func InsertDelivery(ctx context.Context, pool *pgxpool.Pool, subscriptionID uuid.UUID, eventType string, eventID uuid.UUID, payloadHash string, payloadJSON []byte) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `
INSERT INTO integrations.webhook_deliveries (
    subscription_id, event_type, event_id, payload_hash, status, last_response
) VALUES ($1, $2, $3, $4, 'pending', $5)
RETURNING id
`, subscriptionID, eventType, eventID, payloadHash, string(payloadJSON)).Scan(&id)
	return id, err
}

// ListDeliveries returns delivery log rows for a subscription.
func ListDeliveries(ctx context.Context, pool *pgxpool.Pool, subscriptionID uuid.UUID, limit int) ([]Delivery, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, subscription_id, event_type, event_id, payload_hash, attempt_count, status,
       last_http_status, last_response, latency_ms, next_retry_at, delivered_at, created_at
FROM integrations.webhook_deliveries
WHERE subscription_id = $1
ORDER BY created_at DESC
LIMIT $2
`, subscriptionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.SubscriptionID, &d.EventType, &d.EventID, &d.PayloadHash,
			&d.AttemptCount, &d.Status, &d.LastHTTPStatus, &d.LastResponse, &d.LatencyMS,
			&d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDeliveryPayload loads stored payload for a pending delivery.
func GetDeliveryPayload(ctx context.Context, pool *pgxpool.Pool, deliveryID int64) ([]byte, error) {
	var payload *string
	err := pool.QueryRow(ctx, `
SELECT last_response FROM integrations.webhook_deliveries
WHERE id = $1 AND status IN ('pending', 'failed')
`, deliveryID).Scan(&payload)
	if err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, nil
	}
	return []byte(*payload), nil
}

// ListDueDeliveries returns deliveries ready for retry.
func ListDueDeliveries(ctx context.Context, pool *pgxpool.Pool, limit int, now time.Time) ([]Delivery, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT d.id, d.subscription_id, d.event_type, d.event_id, d.payload_hash, d.attempt_count, d.status,
       d.last_http_status, d.last_response, d.latency_ms, d.next_retry_at, d.delivered_at, d.created_at
FROM integrations.webhook_deliveries d
JOIN integrations.webhook_subscriptions s ON s.id = d.subscription_id
WHERE d.status IN ('pending', 'failed')
  AND s.active = true AND s.paused_at IS NULL
  AND (d.next_retry_at IS NULL OR d.next_retry_at <= $1)
ORDER BY d.created_at
LIMIT $2
FOR UPDATE OF d SKIP LOCKED
`, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.SubscriptionID, &d.EventType, &d.EventID, &d.PayloadHash,
			&d.AttemptCount, &d.Status, &d.LastHTTPStatus, &d.LastResponse, &d.LatencyMS,
			&d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// MarkDelivered records a successful delivery.
func MarkDelivered(ctx context.Context, pool *pgxpool.Pool, deliveryID int64, at time.Time, httpStatus, latencyMS int, responseSnippet string) error {
	_, err := pool.Exec(ctx, `
UPDATE integrations.webhook_deliveries
SET status = 'delivered', attempt_count = attempt_count + 1, last_http_status = $2,
    last_response = $3, latency_ms = $4, delivered_at = $5, next_retry_at = NULL
WHERE id = $1
`, deliveryID, httpStatus, truncate(responseSnippet, 1024), latencyMS, at.UTC())
	return err
}

// MarkFailed schedules retry or dead-letters the delivery.
func MarkFailed(ctx context.Context, pool *pgxpool.Pool, deliveryID int64, attempts int, nextRetry time.Time, dead bool, httpStatus int, errMsg string, payloadJSON []byte) error {
	status := "failed"
	if dead {
		status = "dead_lettered"
	}
	var next *time.Time
	if !dead {
		t := nextRetry.UTC()
		next = &t
	}
	_, err := pool.Exec(ctx, `
UPDATE integrations.webhook_deliveries
SET status = $2, attempt_count = $3, next_retry_at = $4, last_http_status = $5,
    last_response = $6
WHERE id = $1
`, deliveryID, status, attempts, next, httpStatus, truncate(errMsg, 1024))
	return err
}

// PurgeOldDeliveries removes delivery log entries older than retention.
func PurgeOldDeliveries(ctx context.Context, pool *pgxpool.Pool, before time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM integrations.webhook_deliveries WHERE created_at < $1
`, before.UTC())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
