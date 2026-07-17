package transcripts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationChannel is email | push | in_app.
type NotificationChannel string

const (
	NotifyChannelEmail NotificationChannel = "email"
	NotifyChannelPush  NotificationChannel = "push"
	NotifyChannelInApp NotificationChannel = "in_app"
)

// NotificationLogRow is one idempotent send record (T10).
type NotificationLogRow struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ItemID    *uuid.UUID
	Event     string
	Channel   NotificationChannel
	Recipient string
	SentAt    time.Time
}

// TryClaimNotification inserts a ledger row. Returns true when this caller owns the send
// (first claim). Returns false when the (order, item, event, channel) was already claimed.
func TryClaimNotification(
	ctx context.Context,
	pool *pgxpool.Pool,
	orderID uuid.UUID,
	itemID *uuid.UUID,
	event, channel, recipient string,
) (bool, error) {
	event = strings.TrimSpace(event)
	channel = strings.TrimSpace(channel)
	recipient = strings.TrimSpace(recipient)
	if pool == nil || orderID == uuid.Nil || event == "" || channel == "" || recipient == "" {
		return false, errors.New("notification claim: missing required fields")
	}
	itemKey := uuid.Nil
	if itemID != nil {
		itemKey = *itemID
	}
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO transcripts.notification_log (order_id, item_id, item_key, event, channel, recipient)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (order_id, item_key, event, channel) DO NOTHING
RETURNING id
`, orderID, itemID, itemKey, event, channel, recipient).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListNotificationLog returns send audit rows for an order (newest first).
func ListNotificationLog(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) ([]NotificationLogRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, order_id, item_id, event, channel, recipient, sent_at
FROM transcripts.notification_log
WHERE order_id = $1
ORDER BY sent_at DESC
`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NotificationLogRow
	for rows.Next() {
		var r NotificationLogRow
		var ch string
		if err := rows.Scan(&r.ID, &r.OrderID, &r.ItemID, &r.Event, &ch, &r.Recipient, &r.SentAt); err != nil {
			return nil, err
		}
		r.Channel = NotificationChannel(ch)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListOrgAdminUserIDs returns active org_admin user ids for registrar exception fan-out.
func ListOrgAdminUserIDs(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, limit int) ([]uuid.UUID, error) {
	if pool == nil || orgID == uuid.Nil {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := pool.Query(ctx, `
SELECT DISTINCT g.user_id
FROM "user".org_role_grants g
WHERE g.org_id = $1
  AND g.role = 'org_admin'
  AND g.org_unit_id IS NULL
  AND (g.expires_at IS NULL OR g.expires_at > NOW())
ORDER BY g.user_id
LIMIT $2
`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
