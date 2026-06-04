// Package sis provides data access for SIS integration connections and sync logs (plan 13.7).
package sis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Vendor constants.
const (
	VendorPowerSchool    = "powerschool"
	VendorInfiniteCampus = "infinite_campus"
	VendorSkyward        = "skyward"
	VendorAeries         = "aeries"
)

// SyncStatus values.
const (
	SyncStatusRunning = "running"
	SyncStatusSuccess = "success"
	SyncStatusPartial = "partial"
	SyncStatusFailed  = "failed"
)

// Connection is a SIS vendor connection config for an org.
type Connection struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	Vendor          string
	BaseURL         string
	ClientIDRef     string
	ClientSecretRef string
	SyncSchedule    string
	SyncMode        string
	Active          bool
	LastSyncAt      *time.Time
	CreatedAt       time.Time
}

// SyncLog is a record of one sync run for a SIS connection.
type SyncLog struct {
	ID           uuid.UUID
	ConnectionID uuid.UUID
	StartedAt    time.Time
	FinishedAt   *time.Time
	Status       string
	Summary      map[string]int
	Errors       []SyncError
}

// SyncError is one error entry in a sync log.
type SyncError struct {
	RecordID string `json:"record_id"`
	Message  string `json:"message"`
}

// SyncSummary holds itemized counts for a sync run.
type SyncSummary struct {
	UsersCreated       int `json:"users_created"`
	UsersUpdated       int `json:"users_updated"`
	EnrollmentsCreated int `json:"enrollments_created"`
	EnrollmentsUpdated int `json:"enrollments_updated"`
	CoursesCreated     int `json:"courses_created"`
	CoursesUpdated     int `json:"courses_updated"`
	Deactivated        int `json:"deactivated"`
	Skipped            int `json:"skipped"`
	Errored            int `json:"errored"`
}

// ListConnections returns all SIS connections for an org.
func ListConnections(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Connection, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, vendor, base_url, client_id_ref, client_secret_ref,
       sync_schedule, sync_mode, active, last_sync_at, created_at
FROM sis.sis_connections
WHERE org_id = $1
ORDER BY created_at ASC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Connection
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Vendor, &c.BaseURL, &c.ClientIDRef,
			&c.ClientSecretRef, &c.SyncSchedule, &c.SyncMode, &c.Active,
			&c.LastSyncAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetConnection returns a single connection by ID and org.
func GetConnection(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) (*Connection, error) {
	var c Connection
	err := pool.QueryRow(ctx, `
SELECT id, org_id, vendor, base_url, client_id_ref, client_secret_ref,
       sync_schedule, sync_mode, active, last_sync_at, created_at
FROM sis.sis_connections
WHERE id = $1 AND org_id = $2
`, id, orgID).Scan(
		&c.ID, &c.OrgID, &c.Vendor, &c.BaseURL, &c.ClientIDRef,
		&c.ClientSecretRef, &c.SyncSchedule, &c.SyncMode, &c.Active,
		&c.LastSyncAt, &c.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateConnection inserts a new SIS connection.
func CreateConnection(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID,
	vendor, baseURL, clientIDRef, clientSecretRef, syncSchedule, syncMode string,
) (*Connection, error) {
	var c Connection
	err := pool.QueryRow(ctx, `
INSERT INTO sis.sis_connections
    (org_id, vendor, base_url, client_id_ref, client_secret_ref, sync_schedule, sync_mode)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, vendor, base_url, client_id_ref, client_secret_ref,
          sync_schedule, sync_mode, active, last_sync_at, created_at
`, orgID, vendor, baseURL, clientIDRef, clientSecretRef, syncSchedule, syncMode).Scan(
		&c.ID, &c.OrgID, &c.Vendor, &c.BaseURL, &c.ClientIDRef,
		&c.ClientSecretRef, &c.SyncSchedule, &c.SyncMode, &c.Active,
		&c.LastSyncAt, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateConnectionFields contains the patchable fields for a SIS connection.
type UpdateConnectionFields struct {
	BaseURL         *string
	ClientIDRef     *string
	ClientSecretRef *string
	SyncSchedule    *string
	SyncMode        *string
	Active          *bool
}

// UpdateConnection applies non-nil fields to an existing connection.
func UpdateConnection(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID, f UpdateConnectionFields) (*Connection, error) {
	_, err := pool.Exec(ctx, `
UPDATE sis.sis_connections SET
    base_url          = COALESCE($3, base_url),
    client_id_ref     = COALESCE($4, client_id_ref),
    client_secret_ref = COALESCE($5, client_secret_ref),
    sync_schedule     = COALESCE($6, sync_schedule),
    sync_mode         = COALESCE($7, sync_mode),
    active            = COALESCE($8, active)
WHERE id = $1 AND org_id = $2
`, id, orgID, f.BaseURL, f.ClientIDRef, f.ClientSecretRef, f.SyncSchedule, f.SyncMode, f.Active)
	if err != nil {
		return nil, err
	}
	return GetConnection(ctx, pool, orgID, id)
}

// CreateSyncLog inserts a new sync log entry with status "running".
func CreateSyncLog(ctx context.Context, pool *pgxpool.Pool, connectionID uuid.UUID) (*SyncLog, error) {
	var l SyncLog
	err := pool.QueryRow(ctx, `
INSERT INTO sis.sis_sync_logs (connection_id, started_at, status)
VALUES ($1, now(), 'running')
RETURNING id, connection_id, started_at, finished_at, status, summary, errors
`, connectionID).Scan(
		&l.ID, &l.ConnectionID, &l.StartedAt, &l.FinishedAt, &l.Status, nil, nil,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// FinishSyncLog marks a sync log complete with its results.
func FinishSyncLog(ctx context.Context, pool *pgxpool.Pool, logID uuid.UUID, status string, summary SyncSummary, errs []SyncError) error {
	summaryJSON, _ := json.Marshal(summary)
	errsJSON, _ := json.Marshal(errs)
	_, err := pool.Exec(ctx, `
UPDATE sis.sis_sync_logs
SET status = $2, finished_at = now(), summary = $3, errors = $4
WHERE id = $1
`, logID, status, summaryJSON, errsJSON)
	return err
}

// TouchLastSyncAt updates the last_sync_at timestamp on a connection.
func TouchLastSyncAt(ctx context.Context, pool *pgxpool.Pool, connectionID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE sis.sis_connections SET last_sync_at = now() WHERE id = $1
`, connectionID)
	return err
}

// ListSyncLogs returns paginated sync logs for an org's connections.
func ListSyncLogs(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, limit, offset int) ([]SyncLog, error) {
	rows, err := pool.Query(ctx, `
SELECT l.id, l.connection_id, l.started_at, l.finished_at, l.status, l.summary, l.errors
FROM sis.sis_sync_logs l
JOIN sis.sis_connections c ON c.id = l.connection_id
WHERE c.org_id = $1
ORDER BY l.started_at DESC
LIMIT $2 OFFSET $3
`, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSyncLogs(rows)
}

// ListSyncLogsForConnection returns sync logs for a specific connection.
func ListSyncLogsForConnection(ctx context.Context, pool *pgxpool.Pool, connectionID uuid.UUID, limit int) ([]SyncLog, error) {
	rows, err := pool.Query(ctx, `
SELECT id, connection_id, started_at, finished_at, status, summary, errors
FROM sis.sis_sync_logs
WHERE connection_id = $1
ORDER BY started_at DESC
LIMIT $2
`, connectionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSyncLogs(rows)
}

// ListActiveConnections returns all active connections for scheduled sync.
func ListActiveConnections(ctx context.Context, pool *pgxpool.Pool) ([]Connection, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, vendor, base_url, client_id_ref, client_secret_ref,
       sync_schedule, sync_mode, active, last_sync_at, created_at
FROM sis.sis_connections
WHERE active = true
ORDER BY created_at ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Connection
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.OrgID, &c.Vendor, &c.BaseURL, &c.ClientIDRef,
			&c.ClientSecretRef, &c.SyncSchedule, &c.SyncMode, &c.Active,
			&c.LastSyncAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanSyncLogs(rows pgx.Rows) ([]SyncLog, error) {
	var out []SyncLog
	for rows.Next() {
		var l SyncLog
		var summaryRaw, errorsRaw []byte
		if err := rows.Scan(&l.ID, &l.ConnectionID, &l.StartedAt, &l.FinishedAt,
			&l.Status, &summaryRaw, &errorsRaw); err != nil {
			return nil, err
		}
		if summaryRaw != nil {
			var s map[string]int
			_ = json.Unmarshal(summaryRaw, &s)
			l.Summary = s
		}
		if errorsRaw != nil {
			var e []SyncError
			_ = json.Unmarshal(errorsRaw, &e)
			l.Errors = e
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
