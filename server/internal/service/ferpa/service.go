// Package ferpa implements FERPA-compliance enforcement: directory opt-out filtering,
// legitimate-educational-interest (LEI) gating, and disclosure audit logging (plan 10.1).
package ferpa

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/ferpa"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// LEIPermission is the RBAC permission that grants legitimate educational interest
// to read a student's detailed education record (34 CFR § 99.34).
const LEIPermission = "compliance:records:read:*"

// AdminPermission gates FERPA admin actions (approve/deny requests, read disclosure log).
const AdminPermission = "compliance:ferpa:admin:*"

// CheckAdmin returns true when the given user holds the compliance:ferpa:admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// IsDirectoryOptOut returns whether the student has opted out of directory-information sharing.
func IsDirectoryOptOut(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) (bool, error) {
	return repo.DirectoryOptOut(ctx, pool, studentID)
}

// SetDirectoryOptOut persists the opt-out flag for the student.
func SetDirectoryOptOut(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, optOut bool) error {
	return repo.SetDirectoryOptOut(ctx, pool, studentID, optOut)
}

// LogDisclosure writes a FERPA disclosure audit entry. It is a best-effort call;
// errors are returned to callers so they may decide whether to surface them.
func LogDisclosure(ctx context.Context, pool *pgxpool.Pool, orgID, accessorID, studentID uuid.UUID, dataType, authorityClaim string, recipient *string) error {
	return repo.InsertDisclosure(ctx, pool, orgID, accessorID, studentID, dataType, authorityClaim, recipient)
}

// SubmitRecordRequest creates a new FERPA record-access or amendment request.
func SubmitRecordRequest(ctx context.Context, pool *pgxpool.Pool, orgID, studentID, requesterID uuid.UUID, requestType, notes string, amendmentField, amendmentValue *string) (uuid.UUID, error) {
	return repo.InsertRecordRequest(ctx, pool, orgID, studentID, requesterID, requestType, notes, amendmentField, amendmentValue)
}

// ListRecordRequests returns pending FERPA record-access requests for the org.
func ListRecordRequests(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]repo.RecordRequest, error) {
	return repo.ListRecordRequestsByOrg(ctx, pool, orgID, 200)
}

// UpdateRecordRequest updates the status of a record request. Also logs a disclosure
// when transitioning to "approved" or "completed".
func UpdateRecordRequest(ctx context.Context, pool *pgxpool.Pool, id, adminID, orgID uuid.UUID, status string, notes, archivePath *string) error {
	req, err := repo.GetRecordRequest(ctx, pool, id)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrNotFound
	}
	if err := repo.UpdateRecordRequestStatus(ctx, pool, id, status, notes, archivePath); err != nil {
		return err
	}
	if status == "approved" || status == "completed" {
		_ = repo.InsertDisclosure(ctx, pool, orgID, adminID, req.StudentID,
			"record_request", "school_official", nil)
	}
	return nil
}

// GrantConsent stores a third-party disclosure consent record.
func GrantConsent(ctx context.Context, pool *pgxpool.Pool, orgID, studentID, grantedBy uuid.UUID, recipient, purpose string, dataFields []string, expiresAt *time.Time) (uuid.UUID, error) {
	id, err := repo.InsertConsent(ctx, pool, orgID, studentID, grantedBy, recipient, purpose, dataFields, expiresAt)
	if err != nil {
		return uuid.UUID{}, err
	}
	_ = repo.InsertDisclosure(ctx, pool, orgID, grantedBy, studentID, "consent_grant", "consent", &recipient)
	return id, nil
}

// RevokeConsent revokes a consent record. Returns ErrNotFound when the record
// does not belong to the given student or is already revoked.
func RevokeConsent(ctx context.Context, pool *pgxpool.Pool, id, studentID uuid.UUID) error {
	ok, err := repo.RevokeConsent(ctx, pool, id, studentID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// ListDisclosures returns disclosure log entries for an org within a time window.
func ListDisclosures(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, from, to time.Time) ([]repo.DisclosureLogEntry, error) {
	return repo.ListDisclosures(ctx, pool, orgID, from, to, 5000)
}

// ErrNotFound signals a missing or inaccessible FERPA record.
var ErrNotFound = errNotFound("ferpa record not found")

type errNotFound string

func (e errNotFound) Error() string { return string(e) }
