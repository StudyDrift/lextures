// Package broadcasts provides data access for district/school broadcast messages (plan 13.10).
package broadcasts

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Broadcast is a single district or school broadcast message.
type Broadcast struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	SchoolID    *uuid.UUID
	SenderID    uuid.UUID
	Type        string
	Audience    json.RawMessage
	Subject     string
	Body        string
	ScheduledAt *time.Time
	SentAt      *time.Time
	Status      string
	CreatedAt   time.Time
}

// CreateParams holds fields for inserting a new broadcast.
type CreateParams struct {
	OrgID       uuid.UUID
	SchoolID    *uuid.UUID
	SenderID    uuid.UUID
	Type        string
	Audience    json.RawMessage
	Subject     string
	Body        string
	ScheduledAt *time.Time
	Status      string
}

// Create inserts a new broadcast and immediately marks sent if no scheduled time.
func Create(ctx context.Context, pool *pgxpool.Pool, p CreateParams) (*Broadcast, error) {
	if len(p.Audience) == 0 {
		p.Audience = json.RawMessage("{}")
	}
	var sentAt *time.Time
	status := p.Status
	if status == "" {
		status = "draft"
	}
	if p.ScheduledAt == nil && status == "sent" {
		now := time.Now().UTC()
		sentAt = &now
	}
	var b Broadcast
	err := pool.QueryRow(ctx, `
INSERT INTO broadcast.broadcasts
    (org_id, school_id, sender_id, type, audience, subject, body, scheduled_at, sent_at, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, org_id, school_id, sender_id, type, audience, subject, body,
          scheduled_at, sent_at, status, created_at
`, p.OrgID, p.SchoolID, p.SenderID, p.Type, []byte(p.Audience), p.Subject, p.Body,
		p.ScheduledAt, sentAt, status).Scan(
		&b.ID, &b.OrgID, &b.SchoolID, &b.SenderID, &b.Type, &b.Audience, &b.Subject, &b.Body,
		&b.ScheduledAt, &b.SentAt, &b.Status, &b.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// ListByOrg returns recent broadcasts in an org, most recent first.
func ListByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, limit int) ([]Broadcast, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, org_id, school_id, sender_id, type, audience, subject, body,
       scheduled_at, sent_at, status, created_at
FROM broadcast.broadcasts
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2
`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBroadcasts(rows)
}

// Get returns a single broadcast scoped to an org, or (nil, nil) if not found.
func Get(ctx context.Context, pool *pgxpool.Pool, orgID, broadcastID uuid.UUID) (*Broadcast, error) {
	var b Broadcast
	err := pool.QueryRow(ctx, `
SELECT id, org_id, school_id, sender_id, type, audience, subject, body,
       scheduled_at, sent_at, status, created_at
FROM broadcast.broadcasts
WHERE id = $1 AND org_id = $2
`, broadcastID, orgID).Scan(
		&b.ID, &b.OrgID, &b.SchoolID, &b.SenderID, &b.Type, &b.Audience, &b.Subject, &b.Body,
		&b.ScheduledAt, &b.SentAt, &b.Status, &b.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// ListForUser returns broadcasts the user has an in-app receipt for and hasn't acknowledged
// (emergency) or that are still active in the user's banner queue.
func ListForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Broadcast, error) {
	rows, err := pool.Query(ctx, `
SELECT b.id, b.org_id, b.school_id, b.sender_id, b.type, b.audience, b.subject, b.body,
       b.scheduled_at, b.sent_at, b.status, b.created_at
FROM broadcast.broadcasts b
JOIN broadcast.broadcast_receipts r
    ON r.broadcast_id = b.id AND r.user_id = $1 AND r.channel = 'in_app'
WHERE b.status = 'sent'
  AND (b.type = 'emergency' AND r.acknowledged_at IS NULL)
ORDER BY b.created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBroadcasts(rows)
}

// EnqueueRecipients inserts in-app receipt rows for every member of the org. The rows act
// as both the delivery log and the per-user banner queue for in-app channels.
func EnqueueRecipients(ctx context.Context, pool *pgxpool.Pool, broadcastID, orgID uuid.UUID) (int, error) {
	tag, err := pool.Exec(ctx, `
INSERT INTO broadcast.broadcast_receipts (broadcast_id, user_id, channel, delivered_at)
SELECT $1, u.id, 'in_app', NOW()
FROM "user".users u
WHERE u.org_id = $2
ON CONFLICT (broadcast_id, user_id, channel) DO NOTHING
`, broadcastID, orgID)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// Acknowledge records that a user acknowledged an emergency broadcast (in_app channel).
func Acknowledge(ctx context.Context, pool *pgxpool.Pool, broadcastID, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO broadcast.broadcast_receipts (broadcast_id, user_id, channel, delivered_at, acknowledged_at)
VALUES ($1, $2, 'in_app', NOW(), NOW())
ON CONFLICT (broadcast_id, user_id, channel)
DO UPDATE SET acknowledged_at = COALESCE(broadcast.broadcast_receipts.acknowledged_at, NOW())
`, broadcastID, userID)
	return err
}

// DeliveryReport summarises per-channel receipts for a broadcast.
type DeliveryReport struct {
	TotalRecipients int
	Acknowledged    int
	Unacknowledged  []Unacknowledged
}

// Unacknowledged identifies a user who has not yet acknowledged an emergency broadcast.
type Unacknowledged struct {
	UserID      uuid.UUID
	Email       string
	DisplayName *string
}

// GetDeliveryReport returns aggregated delivery stats for a broadcast.
func GetDeliveryReport(ctx context.Context, pool *pgxpool.Pool, broadcastID uuid.UUID) (*DeliveryReport, error) {
	var rpt DeliveryReport
	err := pool.QueryRow(ctx, `
SELECT
    COUNT(*)                                                       AS total,
    COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL)             AS acked
FROM broadcast.broadcast_receipts
WHERE broadcast_id = $1 AND channel = 'in_app'
`, broadcastID).Scan(&rpt.TotalRecipients, &rpt.Acknowledged)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT u.id, u.email, u.display_name
FROM broadcast.broadcast_receipts r
JOIN "user".users u ON u.id = r.user_id
WHERE r.broadcast_id = $1 AND r.channel = 'in_app' AND r.acknowledged_at IS NULL
ORDER BY u.email ASC
LIMIT 500
`, broadcastID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u Unacknowledged
		if err := rows.Scan(&u.UserID, &u.Email, &u.DisplayName); err != nil {
			return nil, err
		}
		rpt.Unacknowledged = append(rpt.Unacknowledged, u)
	}
	return &rpt, rows.Err()
}

func scanBroadcasts(rows pgx.Rows) ([]Broadcast, error) {
	var out []Broadcast
	for rows.Next() {
		var b Broadcast
		if err := rows.Scan(
			&b.ID, &b.OrgID, &b.SchoolID, &b.SenderID, &b.Type, &b.Audience, &b.Subject, &b.Body,
			&b.ScheduledAt, &b.SentAt, &b.Status, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
