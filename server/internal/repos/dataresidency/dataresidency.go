// Package dataresidency provides read/write access to the data residency access log (plan 10.12).
package dataresidency

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccessLogEntry represents a row in compliance.data_residency_access_log.
type AccessLogEntry struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	OrgRegion     string
	RequestedFrom string
	EventType     string
	RequestPath   *string
	ActorID       *uuid.UUID
	CreatedAt     time.Time
}

// ListAccessLog returns recent access log entries, newest first (up to limit).
func ListAccessLog(ctx context.Context, pool *pgxpool.Pool, limit, offset int32) ([]AccessLogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := pool.Query(ctx, `
SELECT id, org_id, org_region, requested_from, event_type, request_path, actor_id, created_at
FROM compliance.data_residency_access_log
ORDER BY created_at DESC
LIMIT $1 OFFSET $2
`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

// ListAccessLogByOrg returns access log entries for a specific org, newest first.
func ListAccessLogByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, limit, offset int32) ([]AccessLogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := pool.Query(ctx, `
SELECT id, org_id, org_region, requested_from, event_type, request_path, actor_id, created_at
FROM compliance.data_residency_access_log
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func scanEntries(rows pgx.Rows) ([]AccessLogEntry, error) {
	var out []AccessLogEntry
	for rows.Next() {
		var e AccessLogEntry
		if err := rows.Scan(&e.ID, &e.OrgID, &e.OrgRegion, &e.RequestedFrom, &e.EventType, &e.RequestPath, &e.ActorID, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
