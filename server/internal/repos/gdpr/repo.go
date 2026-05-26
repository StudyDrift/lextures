// Package gdpr persists GDPR consent records, DSAR requests, and RoPA entries (plan 10.3).
package gdpr

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConsentRecord is one row from compliance.gdpr_consents.
type ConsentRecord struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Purpose        string
	LawfulBasis    string
	ConsentVersion string
	GrantedAt      time.Time
	WithdrawnAt    *time.Time
	IPHash         *string
	CreatedAt      time.Time
}

// InsertConsent stores a new consent grant for the given user and purpose.
func InsertConsent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, purpose, lawfulBasis, version string, ipHash *string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.gdpr_consents (user_id, purpose, lawful_basis, consent_version, ip_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING id
`, userID, purpose, lawfulBasis, version, ipHash).Scan(&id)
	return id, err
}

// WithdrawConsent sets withdrawn_at on the matching active consent row.
// Returns false when no active consent matching the id+user_id was found.
func WithdrawConsent(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE compliance.gdpr_consents
   SET withdrawn_at = NOW()
 WHERE id = $1
   AND user_id = $2
   AND withdrawn_at IS NULL
`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ListConsents returns all consent records for a user (active and withdrawn).
func ListConsents(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]ConsentRecord, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, purpose, lawful_basis, consent_version,
       granted_at, withdrawn_at, ip_hash, created_at
  FROM compliance.gdpr_consents
 WHERE user_id = $1
 ORDER BY created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ConsentRecord
	for rows.Next() {
		var r ConsentRecord
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.Purpose, &r.LawfulBasis, &r.ConsentVersion,
			&r.GrantedAt, &r.WithdrawnAt, &r.IPHash, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// HasActiveConsent returns true when the user has at least one non-withdrawn consent for purpose.
func HasActiveConsent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, purpose string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM compliance.gdpr_consents
   WHERE user_id = $1
     AND purpose = $2
     AND withdrawn_at IS NULL
)
`, userID, purpose).Scan(&exists)
	return exists, err
}

// DSARRequest is one row from compliance.dsar_requests.
type DSARRequest struct {
	ID              uuid.UUID
	OrgID           *uuid.UUID
	UserID          uuid.UUID
	RequestType     string
	Status          string
	ArchiveURL      *string
	ArchiveExpiresAt *time.Time
	RejectionReason *string
	RequestedAt     time.Time
	DueAt           time.Time
	CompletedAt     *time.Time
	ActionedBy      *uuid.UUID
}

// InsertDSARRequest creates a new DSAR row.
func InsertDSARRequest(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, userID uuid.UUID, requestType string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.dsar_requests (org_id, user_id, request_type)
VALUES ($1, $2, $3)
RETURNING id
`, orgID, userID, requestType).Scan(&id)
	return id, err
}

// GetDSARRequest returns a single request by ID.
func GetDSARRequest(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*DSARRequest, error) {
	r, err := scanDSAR(pool.QueryRow(ctx, `
SELECT id, org_id, user_id, request_type, status,
       archive_url, archive_expires_at, rejection_reason,
       requested_at, due_at, completed_at, actioned_by
  FROM compliance.dsar_requests
 WHERE id = $1
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// ListDSARRequestsForUser returns all DSAR requests by a user.
func ListDSARRequestsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]DSARRequest, error) {
	return queryDSARs(ctx, pool, `
SELECT id, org_id, user_id, request_type, status,
       archive_url, archive_expires_at, rejection_reason,
       requested_at, due_at, completed_at, actioned_by
  FROM compliance.dsar_requests
 WHERE user_id = $1
 ORDER BY requested_at DESC
`, userID)
}

// ListDSARRequestsPending returns all pending/in-progress requests (admin queue).
func ListDSARRequestsPending(ctx context.Context, pool *pgxpool.Pool, limit int) ([]DSARRequest, error) {
	return queryDSARs(ctx, pool, `
SELECT id, org_id, user_id, request_type, status,
       archive_url, archive_expires_at, rejection_reason,
       requested_at, due_at, completed_at, actioned_by
  FROM compliance.dsar_requests
 WHERE status IN ('pending','in_progress')
 ORDER BY due_at ASC
 LIMIT $1
`, limit)
}

// UpdateDSARStatus transitions a request to a new status and optionally sets
// archive_url, archive_expires_at, rejection_reason, and actioned_by.
func UpdateDSARStatus(ctx context.Context, pool *pgxpool.Pool, id, actionedBy uuid.UUID, status string, archiveURL *string, archiveExpiresAt *time.Time, rejectionReason *string) error {
	var completedAt *time.Time
	if status == "completed" || status == "rejected" {
		t := time.Now().UTC()
		completedAt = &t
	}
	_, err := pool.Exec(ctx, `
UPDATE compliance.dsar_requests
   SET status            = $2,
       archive_url       = COALESCE($3, archive_url),
       archive_expires_at = COALESCE($4, archive_expires_at),
       rejection_reason  = COALESCE($5, rejection_reason),
       completed_at      = COALESCE($6, completed_at),
       actioned_by       = $7
 WHERE id = $1
`, id, status, archiveURL, archiveExpiresAt, rejectionReason, completedAt, actionedBy)
	return err
}

// CountOverdueDSARs returns the number of pending requests past their due_at.
func CountOverdueDSARs(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM compliance.dsar_requests
 WHERE status IN ('pending','in_progress')
   AND due_at < NOW()
`).Scan(&n)
	return n, err
}

// ListDSARsDueSoon returns pending requests whose due_at is within the given horizon.
func ListDSARsDueSoon(ctx context.Context, pool *pgxpool.Pool, horizon time.Duration) ([]DSARRequest, error) {
	cutoff := time.Now().UTC().Add(horizon)
	return queryDSARs(ctx, pool, `
SELECT id, org_id, user_id, request_type, status,
       archive_url, archive_expires_at, rejection_reason,
       requested_at, due_at, completed_at, actioned_by
  FROM compliance.dsar_requests
 WHERE status IN ('pending','in_progress')
   AND due_at <= $1
   AND due_at > NOW()
 ORDER BY due_at ASC
`, cutoff)
}

// RoPAEntry is one row from compliance.ropa_entries.
type RoPAEntry struct {
	ID              uuid.UUID
	OrgID           uuid.UUID
	ActivityName    string
	Purpose         string
	LawfulBasis     string
	DataCategories  []string
	DataSubjects    []string
	RetentionPeriod string
	SubProcessors   []string
	UpdatedAt       time.Time
}

// InsertRoPAEntry adds a new Record of Processing Activity.
func InsertRoPAEntry(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, activityName, purpose, lawfulBasis, retentionPeriod string, dataCategories, dataSubjects, subProcessors []string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.ropa_entries
  (org_id, activity_name, purpose, lawful_basis, data_categories, data_subjects, retention_period, sub_processors)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id
`, orgID, activityName, purpose, lawfulBasis, dataCategories, dataSubjects, retentionPeriod, subProcessors).Scan(&id)
	return id, err
}

// ListRoPAEntries returns all RoPA entries for an org.
func ListRoPAEntries(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]RoPAEntry, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, activity_name, purpose, lawful_basis,
       data_categories, data_subjects, retention_period, sub_processors, updated_at
  FROM compliance.ropa_entries
 WHERE org_id = $1
 ORDER BY updated_at DESC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RoPAEntry
	for rows.Next() {
		var e RoPAEntry
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.ActivityName, &e.Purpose, &e.LawfulBasis,
			&e.DataCategories, &e.DataSubjects, &e.RetentionPeriod, &e.SubProcessors, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteRoPAEntry removes a RoPA entry by ID.
func DeleteRoPAEntry(ctx context.Context, pool *pgxpool.Pool, id, orgID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM compliance.ropa_entries WHERE id = $1 AND org_id = $2
`, id, orgID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// AnonymiseUser nullifies personally-identifying columns for a user as part of
// an erasure request (Article 17). Financial and audit-log rows are retained as
// tombstones per the retention exception documented in the RoPA.
func AnonymiseUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE "user".users
   SET email          = 'erased-' || id || '@erased.invalid',
       display_name   = 'Erased User',
       first_name     = NULL,
       last_name      = NULL,
       avatar_url     = NULL,
       parent_email   = NULL,
       date_of_birth  = NULL
 WHERE id = $1
`, userID)
	return err
}

func scanDSAR(row pgx.Row) (*DSARRequest, error) {
	var r DSARRequest
	err := row.Scan(
		&r.ID, &r.OrgID, &r.UserID, &r.RequestType, &r.Status,
		&r.ArchiveURL, &r.ArchiveExpiresAt, &r.RejectionReason,
		&r.RequestedAt, &r.DueAt, &r.CompletedAt, &r.ActionedBy,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func queryDSARs(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) ([]DSARRequest, error) {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DSARRequest
	for rows.Next() {
		var r DSARRequest
		if err := rows.Scan(
			&r.ID, &r.OrgID, &r.UserID, &r.RequestType, &r.Status,
			&r.ArchiveURL, &r.ArchiveExpiresAt, &r.RejectionReason,
			&r.RequestedAt, &r.DueAt, &r.CompletedAt, &r.ActionedBy,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
