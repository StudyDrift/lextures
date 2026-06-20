// Package integrations provides database access for inbound third-party
// integration connections and external course links (plan 16.4).
package integrations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a connection or link row does not exist.
var ErrNotFound = errors.New("integrations: not found")

// Connection is one row of integrations.oauth_connections. Token columns hold
// ciphertext (see internal/crypto); callers decrypt on demand.
type Connection struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	Provider        string
	ExternalID      string
	AccessTokenEnc  string
	RefreshTokenEnc string
	TokenExpiresAt  *time.Time
	Scopes          []string
	ConnectedBy     *uuid.UUID
	LastSyncedAt    *time.Time
	LastSyncError   *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// UpsertParams holds the fields needed to create or refresh a connection.
type UpsertParams struct {
	OrgID           uuid.UUID
	Provider        string
	ExternalID      string
	AccessTokenEnc  string
	RefreshTokenEnc string
	TokenExpiresAt  *time.Time
	Scopes          []string
	ConnectedBy     *uuid.UUID
}

const connectionColumns = `id, org_id, provider, external_id, access_token_enc, refresh_token_enc,
	token_expires_at, scopes, connected_by, last_synced_at, last_sync_error, created_at, updated_at`

func scanConnection(row pgx.Row) (Connection, error) {
	var c Connection
	err := row.Scan(&c.ID, &c.OrgID, &c.Provider, &c.ExternalID, &c.AccessTokenEnc, &c.RefreshTokenEnc,
		&c.TokenExpiresAt, &c.Scopes, &c.ConnectedBy, &c.LastSyncedAt, &c.LastSyncError, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// Upsert creates a connection or updates the tokens/scopes of an existing one
// matching (org_id, provider, external_id).
func Upsert(ctx context.Context, pool *pgxpool.Pool, p UpsertParams) (Connection, error) {
	if pool == nil {
		return Connection{}, errors.New("integrations: db pool is nil")
	}
	if p.Scopes == nil {
		p.Scopes = []string{}
	}
	row := pool.QueryRow(ctx, `
INSERT INTO integrations.oauth_connections
	(org_id, provider, external_id, access_token_enc, refresh_token_enc, token_expires_at, scopes, connected_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (org_id, provider, external_id) DO UPDATE SET
	access_token_enc = EXCLUDED.access_token_enc,
	refresh_token_enc = EXCLUDED.refresh_token_enc,
	token_expires_at = EXCLUDED.token_expires_at,
	scopes = EXCLUDED.scopes,
	connected_by = EXCLUDED.connected_by,
	last_sync_error = NULL,
	updated_at = now()
RETURNING `+connectionColumns,
		p.OrgID, p.Provider, p.ExternalID, p.AccessTokenEnc, p.RefreshTokenEnc, p.TokenExpiresAt, p.Scopes, p.ConnectedBy)
	return scanConnection(row)
}

// ListByOrg returns all connections for an org ordered by provider.
func ListByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Connection, error) {
	if pool == nil {
		return nil, errors.New("integrations: db pool is nil")
	}
	rows, err := pool.Query(ctx, `SELECT `+connectionColumns+`
FROM integrations.oauth_connections WHERE org_id = $1 ORDER BY provider, created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Connection, 0)
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get returns one connection scoped to its org. Returns ErrNotFound when absent.
func Get(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) (Connection, error) {
	if pool == nil {
		return Connection{}, errors.New("integrations: db pool is nil")
	}
	row := pool.QueryRow(ctx, `SELECT `+connectionColumns+`
FROM integrations.oauth_connections WHERE id = $1 AND org_id = $2`, id, orgID)
	c, err := scanConnection(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	return c, err
}

// Delete removes a connection scoped to its org. Returns ErrNotFound when absent.
func Delete(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) error {
	if pool == nil {
		return errors.New("integrations: db pool is nil")
	}
	tag, err := pool.Exec(ctx, `DELETE FROM integrations.oauth_connections WHERE id = $1 AND org_id = $2`, id, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkSynced records a successful sync timestamp and clears any prior error.
func MarkSynced(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, at time.Time) error {
	if pool == nil {
		return errors.New("integrations: db pool is nil")
	}
	_, err := pool.Exec(ctx, `UPDATE integrations.oauth_connections
SET last_synced_at = $2, last_sync_error = NULL, updated_at = now() WHERE id = $1`, id, at)
	return err
}

// MarkSyncError records a failure reason without advancing last_synced_at.
func MarkSyncError(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, reason string) error {
	if pool == nil {
		return errors.New("integrations: db pool is nil")
	}
	_, err := pool.Exec(ctx, `UPDATE integrations.oauth_connections
SET last_sync_error = $2, updated_at = now() WHERE id = $1`, id, reason)
	return err
}

// CourseLink is one row of integrations.external_course_links.
type CourseLink struct {
	ID                uuid.UUID
	LexturesCourseID  uuid.UUID
	ConnectionID      uuid.UUID
	ExternalCourseID  string
	SyncRoster        bool
	SyncIntervalHours int16
	LastSyncedAt      *time.Time
	CreatedAt         time.Time
}

// LinkParams holds the fields to create or update a course link.
type LinkParams struct {
	LexturesCourseID  uuid.UUID
	ConnectionID      uuid.UUID
	ExternalCourseID  string
	SyncRoster        bool
	SyncIntervalHours int16
}

const linkColumns = `id, lextures_course_id, connection_id, external_course_id,
	sync_roster, sync_interval_hours, last_synced_at, created_at`

func scanLink(row pgx.Row) (CourseLink, error) {
	var l CourseLink
	err := row.Scan(&l.ID, &l.LexturesCourseID, &l.ConnectionID, &l.ExternalCourseID,
		&l.SyncRoster, &l.SyncIntervalHours, &l.LastSyncedAt, &l.CreatedAt)
	return l, err
}

// UpsertLink creates or updates the link between a Lextures course and external class.
func UpsertLink(ctx context.Context, pool *pgxpool.Pool, p LinkParams) (CourseLink, error) {
	if pool == nil {
		return CourseLink{}, errors.New("integrations: db pool is nil")
	}
	if p.SyncIntervalHours <= 0 {
		p.SyncIntervalHours = 6
	}
	row := pool.QueryRow(ctx, `
INSERT INTO integrations.external_course_links
	(lextures_course_id, connection_id, external_course_id, sync_roster, sync_interval_hours)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (lextures_course_id, connection_id) DO UPDATE SET
	external_course_id = EXCLUDED.external_course_id,
	sync_roster = EXCLUDED.sync_roster,
	sync_interval_hours = EXCLUDED.sync_interval_hours
RETURNING `+linkColumns,
		p.LexturesCourseID, p.ConnectionID, p.ExternalCourseID, p.SyncRoster, p.SyncIntervalHours)
	return scanLink(row)
}

// ListLinksByConnection returns all course links for a connection.
func ListLinksByConnection(ctx context.Context, pool *pgxpool.Pool, connID uuid.UUID) ([]CourseLink, error) {
	if pool == nil {
		return nil, errors.New("integrations: db pool is nil")
	}
	rows, err := pool.Query(ctx, `SELECT `+linkColumns+`
FROM integrations.external_course_links WHERE connection_id = $1 ORDER BY created_at`, connID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CourseLink, 0)
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CurrentCourseEmails returns the lower-cased emails of users currently
// enrolled (active or pending) in a Lextures course, used to diff against an
// imported external roster (plan 16.4 FR-8).
func CurrentCourseEmails(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]string, error) {
	if pool == nil {
		return nil, errors.New("integrations: db pool is nil")
	}
	rows, err := pool.Query(ctx, `
SELECT lower(u.email)
FROM course.course_enrollments ce
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE ce.course_id = $1 AND (ce.active OR ce.invitation_pending)`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		out = append(out, email)
	}
	return out, rows.Err()
}

// MarkLinkSynced records a successful roster sync timestamp on a link.
func MarkLinkSynced(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, at time.Time) error {
	if pool == nil {
		return errors.New("integrations: db pool is nil")
	}
	_, err := pool.Exec(ctx, `UPDATE integrations.external_course_links
SET last_synced_at = $2 WHERE id = $1`, id, at)
	return err
}
