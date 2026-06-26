// Package adminaudit persists and queries the compliance.admin_audit_log table (plan 10.11).
package adminaudit

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event is one row from compliance.admin_audit_log.
type Event struct {
	EventID     uuid.UUID
	OrgID       *uuid.UUID
	EventType   string
	ActorID     uuid.UUID
	ActorIP     *string
	UserAgent   *string
	TargetType  *string
	TargetID    *uuid.UUID
	BeforeValue []byte
	AfterValue  []byte
	ChainHash   *string
	Timestamp   time.Time
}

// InsertParams holds the fields required to write one audit event.
type InsertParams struct {
	OrgID       *uuid.UUID
	EventType   string
	ActorID     uuid.UUID
	ActorIP     *string
	UserAgent   *string
	TargetType  *string
	TargetID    *uuid.UUID
	BeforeValue []byte
	AfterValue  []byte
	ChainHash   *string
}

const insertSQL = `
INSERT INTO compliance.admin_audit_log
  (org_id, event_type, actor_id, actor_ip, user_agent, target_type, target_id,
   before_value, after_value, chain_hash)
VALUES ($1, $2, $3, $4::inet, $5, $6, $7, $8, $9, $10)
RETURNING event_id, "timestamp"
`

// Insert writes a new audit event using the pool (its own implicit transaction).
func Insert(ctx context.Context, pool *pgxpool.Pool, p InsertParams) (uuid.UUID, time.Time, error) {
	var id uuid.UUID
	var ts time.Time
	err := pool.QueryRow(ctx, insertSQL,
		p.OrgID, p.EventType, p.ActorID, p.ActorIP, p.UserAgent,
		p.TargetType, p.TargetID, nullBytes(p.BeforeValue), nullBytes(p.AfterValue), p.ChainHash,
	).Scan(&id, &ts)
	return id, ts, err
}

// Query holds optional filter parameters for List.
type Query struct {
	OrgID     *uuid.UUID
	ActorID   *uuid.UUID
	EventType *string
	TargetID  *uuid.UUID
	From      time.Time
	To        time.Time
	Limit     int
}

// List returns audit events matching the optional filters, ordered by timestamp DESC.
func List(ctx context.Context, pool *pgxpool.Pool, q Query) ([]Event, error) {
	if q.Limit <= 0 {
		q.Limit = 500
	}
	rows, err := pool.Query(ctx, `
SELECT
  event_id, org_id, event_type, actor_id,
  actor_ip::text, user_agent,
  target_type, target_id,
  before_value, after_value, chain_hash, "timestamp"
FROM compliance.admin_audit_log
WHERE
  ($1::uuid IS NULL OR org_id   = $1::uuid)
  AND ($2::uuid IS NULL OR actor_id  = $2::uuid)
  AND ($3::text IS NULL OR event_type = $3::text)
  AND ($4::uuid IS NULL OR target_id  = $4::uuid)
  AND "timestamp" >= $5
  AND "timestamp" <= $6
ORDER BY "timestamp" DESC
LIMIT $7
`, q.OrgID, q.ActorID, q.EventType, q.TargetID, q.From, q.To, q.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.EventID, &e.OrgID, &e.EventType, &e.ActorID,
			&e.ActorIP, &e.UserAgent,
			&e.TargetType, &e.TargetID,
			&e.BeforeValue, &e.AfterValue, &e.ChainHash, &e.Timestamp,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetByID returns a single audit event by its event_id, or (nil, nil) if not found.
func GetByID(ctx context.Context, pool *pgxpool.Pool, eventID uuid.UUID) (*Event, error) {
	var e Event
	err := pool.QueryRow(ctx, `
SELECT
  event_id, org_id, event_type, actor_id,
  actor_ip::text, user_agent,
  target_type, target_id,
  before_value, after_value, chain_hash, "timestamp"
FROM compliance.admin_audit_log
WHERE event_id = $1
`, eventID).Scan(
		&e.EventID, &e.OrgID, &e.EventType, &e.ActorID,
		&e.ActorIP, &e.UserAgent,
		&e.TargetType, &e.TargetID,
		&e.BeforeValue, &e.AfterValue, &e.ChainHash, &e.Timestamp,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &e, err
}

// nullBytes returns nil when b is empty, otherwise b (for JSONB columns).
func nullBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
