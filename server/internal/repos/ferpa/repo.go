// Package ferpa persists FERPA directory opt-out flags, record-access requests,
// third-party consent records, and the disclosure log (plan 10.1).
package ferpa

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DirectoryOptOut returns the ferpa_directory_opt_out flag for the given student.
func DirectoryOptOut(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) (bool, error) {
	var v bool
	err := pool.QueryRow(ctx,
		`SELECT ferpa_directory_opt_out FROM "user".users WHERE id = $1`,
		studentID,
	).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return v, err
}

// SetDirectoryOptOut writes the ferpa_directory_opt_out flag for the given student.
func SetDirectoryOptOut(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, optOut bool) error {
	_, err := pool.Exec(ctx,
		`UPDATE "user".users SET ferpa_directory_opt_out = $2 WHERE id = $1`,
		studentID, optOut,
	)
	return err
}

// RecordRequest is one row from compliance.ferpa_record_requests.
type RecordRequest struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	StudentID      uuid.UUID
	RequesterID    uuid.UUID
	RequestType    string
	Status         string
	AmendmentField *string
	AmendmentValue *string
	Notes          *string
	ArchivePath    *string
	RequestedAt    time.Time
	DueAt          *time.Time
	CompletedAt    *time.Time
}

// InsertRecordRequest creates a new FERPA record-access request.
// due_at is set to requested_at + 45 days per 34 CFR § 99.10.
func InsertRecordRequest(ctx context.Context, pool *pgxpool.Pool, orgID, studentID, requesterID uuid.UUID, requestType, notes string, amendmentField, amendmentValue *string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.ferpa_record_requests
  (org_id, student_id, requester_id, request_type, amendment_field, amendment_value, notes, due_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW() + INTERVAL '45 days')
RETURNING id
`, orgID, studentID, requesterID, requestType, amendmentField, amendmentValue, nullText(notes)).Scan(&id)
	return id, err
}

// GetRecordRequest returns a single record request by ID.
func GetRecordRequest(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*RecordRequest, error) {
	var r RecordRequest
	err := pool.QueryRow(ctx, `
SELECT id, org_id, student_id, requester_id, request_type, status,
       amendment_field, amendment_value, notes, archive_path,
       requested_at, due_at, completed_at
FROM compliance.ferpa_record_requests WHERE id = $1
`, id).Scan(
		&r.ID, &r.OrgID, &r.StudentID, &r.RequesterID, &r.RequestType, &r.Status,
		&r.AmendmentField, &r.AmendmentValue, &r.Notes, &r.ArchivePath,
		&r.RequestedAt, &r.DueAt, &r.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &r, err
}

// ListRecordRequestsByOrg returns pending/active requests for a given org.
func ListRecordRequestsByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, limit int) ([]RecordRequest, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, student_id, requester_id, request_type, status,
       amendment_field, amendment_value, notes, archive_path,
       requested_at, due_at, completed_at
FROM compliance.ferpa_record_requests
WHERE org_id = $1
ORDER BY requested_at DESC
LIMIT $2
`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RecordRequest
	for rows.Next() {
		var r RecordRequest
		if err := rows.Scan(
			&r.ID, &r.OrgID, &r.StudentID, &r.RequesterID, &r.RequestType, &r.Status,
			&r.AmendmentField, &r.AmendmentValue, &r.Notes, &r.ArchivePath,
			&r.RequestedAt, &r.DueAt, &r.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateRecordRequestStatus updates status, optional notes, and optional archive_path.
func UpdateRecordRequestStatus(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, notes, archivePath *string) error {
	var completedAt *time.Time
	if status == "completed" {
		now := time.Now().UTC()
		completedAt = &now
	}
	_, err := pool.Exec(ctx, `
UPDATE compliance.ferpa_record_requests
SET status       = $2,
    notes        = COALESCE($3, notes),
    archive_path = COALESCE($4, archive_path),
    completed_at = COALESCE($5, completed_at)
WHERE id = $1
`, id, status, notes, archivePath, completedAt)
	return err
}

// ConsentRecord is one row from compliance.ferpa_consent_records.
type ConsentRecord struct {
	ID          uuid.UUID
	OrgID       uuid.UUID
	StudentID   uuid.UUID
	GrantedBy   uuid.UUID
	Recipient   string
	Purpose     string
	DataFields  []string
	ConsentedAt time.Time
	ExpiresAt   *time.Time
	RevokedAt   *time.Time
}

// InsertConsent creates a new third-party consent record.
func InsertConsent(ctx context.Context, pool *pgxpool.Pool, orgID, studentID, grantedBy uuid.UUID, recipient, purpose string, dataFields []string, expiresAt *time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.ferpa_consent_records
  (org_id, student_id, granted_by, recipient, purpose, data_fields, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id
`, orgID, studentID, grantedBy, recipient, purpose, dataFields, expiresAt).Scan(&id)
	return id, err
}

// RevokeConsent sets revoked_at on a consent record. Returns false when the record
// is not found or does not belong to the given student.
func RevokeConsent(ctx context.Context, pool *pgxpool.Pool, id, studentID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE compliance.ferpa_consent_records
SET revoked_at = NOW()
WHERE id = $1 AND student_id = $2 AND revoked_at IS NULL
`, id, studentID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DisclosureLogEntry is one row from compliance.ferpa_disclosure_log.
type DisclosureLogEntry struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	AccessorID     uuid.UUID
	StudentID      uuid.UUID
	DataType       string
	AuthorityClaim string
	Recipient      *string
	LoggedAt       time.Time
}

// InsertDisclosure writes an audit entry to the FERPA disclosure log.
func InsertDisclosure(ctx context.Context, pool *pgxpool.Pool, orgID, accessorID, studentID uuid.UUID, dataType, authorityClaim string, recipient *string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO compliance.ferpa_disclosure_log
  (org_id, accessor_id, student_id, data_type, authority_claim, recipient)
VALUES ($1, $2, $3, $4, $5, $6)
`, orgID, accessorID, studentID, dataType, authorityClaim, recipient)
	return err
}

// ListDisclosures returns disclosure log entries for a given org within a time window.
func ListDisclosures(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, from, to time.Time, limit int) ([]DisclosureLogEntry, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, accessor_id, student_id, data_type, authority_claim, recipient, logged_at
FROM compliance.ferpa_disclosure_log
WHERE org_id = $1 AND logged_at >= $2 AND logged_at <= $3
ORDER BY logged_at DESC
LIMIT $4
`, orgID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DisclosureLogEntry
	for rows.Next() {
		var e DisclosureLogEntry
		if err := rows.Scan(&e.ID, &e.OrgID, &e.AccessorID, &e.StudentID, &e.DataType, &e.AuthorityClaim, &e.Recipient, &e.LoggedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
