// Package coppa implements the COPPA verifiable parental consent workflow (plan 10.2).
// References: 15 U.S.C. §§ 6501–6506; FTC Rule 16 CFR Part 312.
package coppa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	appcrypto "github.com/lextures/lextures/server/internal/crypto"
)

// ConsentStatus mirrors the coppa_consent_status CHECK constraint.
type ConsentStatus string

const (
	ConsentStatusNotRequired ConsentStatus = "not_required"
	ConsentStatusPending     ConsentStatus = "pending"
	ConsentStatusApproved    ConsentStatus = "approved"
	ConsentStatusRevoked     ConsentStatus = "revoked"
)

// ConsentMethod mirrors the consent_method CHECK constraint.
type ConsentMethod string

const (
	ConsentMethodEmailSigned         ConsentMethod = "email_signed"
	ConsentMethodSchoolAuthorization ConsentMethod = "school_authorization"
	ConsentMethodUpload              ConsentMethod = "upload"
	ConsentMethodDirect              ConsentMethod = "direct"
)

// ConsentToken is returned when a new signed-link consent is initiated.
type ConsentToken struct {
	// RawToken is sent to the parent via email — never stored.
	RawToken string
	// ConsentID is the compliance.coppa_consents row ID.
	ConsentID uuid.UUID
}

// ConsentRecord represents a row in compliance.coppa_consents.
type ConsentRecord struct {
	ID                uuid.UUID
	OrgID             uuid.UUID
	StudentID         uuid.UUID
	ParentEmail       string
	ConsentMethod     ConsentMethod
	ConsentedAt       *time.Time
	RevokedAt         *time.Time
	PriorRecordID     *uuid.UUID
	AIFeaturesEnabled bool
	CreatedAt         time.Time
}

// UserConsentStatus is the COPPA status of a user account.
type UserConsentStatus struct {
	CoppaMinor        bool
	ConsentStatus     ConsentStatus
	ParentEmail       *string
	AIFeaturesEnabled bool
}

// BulkImportRow is one row from a district CSV bulk import.
type BulkImportRow struct {
	StudentID     uuid.UUID
	ParentEmail   string
	ConsentDate   time.Time
	ConsentMethod ConsentMethod
}

// BulkImportResult summarises a bulk import operation.
type BulkImportResult struct {
	Imported int
	Skipped  int
	Errors   []string
}

var (
	ErrNotFound        = errors.New("coppa: record not found")
	ErrAlreadyApproved = errors.New("coppa: consent already approved")
	ErrTokenExpired    = errors.New("coppa: consent token expired")
	ErrTokenInvalid    = errors.New("coppa: invalid consent token")
	ErrNotMinor        = errors.New("coppa: user is not a coppa_minor")
	ErrAlreadyRevoked  = errors.New("coppa: consent already revoked")
)

// consentTokenTTL is the validity window for email consent links (16 CFR §312.7 security).
const consentTokenTTL = 72 * time.Hour

// ClassifyMinor returns true when date of birth indicates the user is under 13.
func ClassifyMinor(dob time.Time, now time.Time) bool {
	age := now.Year() - dob.Year()
	if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
		age--
	}
	return age < 13
}

// FlagMinorAccount updates users.coppa_minor and sets consent_status to 'pending'
// when the DOB implies age < 13, writing parent_email and date_of_birth at the same time.
func FlagMinorAccount(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, dob time.Time, parentEmail string) error {
	isMinor := ClassifyMinor(dob, time.Now().UTC())
	status := ConsentStatusNotRequired
	if isMinor {
		status = ConsentStatusPending
	}
	encParentEmail, err := appcrypto.EncryptString(strings.ToLower(strings.TrimSpace(parentEmail)))
	if err != nil {
		return err
	}
	encDOB, err := appcrypto.EncryptString(dob.Format("2006-01-02"))
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
UPDATE "user".users
   SET coppa_minor          = $2,
       coppa_consent_status = $3,
       parent_email         = $4,
       date_of_birth        = $5
 WHERE id = $1`,
		userID, isMinor, string(status), encParentEmail, encDOB,
	)
	return err
}

// GetUserConsentStatus returns the COPPA status fields for a user.
func GetUserConsentStatus(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*UserConsentStatus, error) {
	var s UserConsentStatus
	var status string
	var parentEmail *string
	var aiFeaturesEnabled bool
	err := pool.QueryRow(ctx, `
SELECT coppa_minor, coppa_consent_status, parent_email
  FROM "user".users
 WHERE id = $1`, userID).Scan(&s.CoppaMinor, &status, &parentEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	s.ConsentStatus = ConsentStatus(status)
	decryptedParentEmail, err := appcrypto.MaybeDecryptString(parentEmail)
	if err != nil {
		return nil, err
	}
	s.ParentEmail = decryptedParentEmail

	// Pull AI opt-in from the active consent record if approved.
	if s.ConsentStatus == ConsentStatusApproved {
		_ = pool.QueryRow(ctx, `
SELECT ai_features_enabled
  FROM compliance.coppa_consents
 WHERE student_id = $1
   AND revoked_at IS NULL
   AND consented_at IS NOT NULL
 ORDER BY created_at DESC
 LIMIT 1`, userID).Scan(&aiFeaturesEnabled)
	}
	s.AIFeaturesEnabled = aiFeaturesEnabled
	return &s, nil
}

// IsCoppaMinorBlocked returns true when the user is a coppa_minor whose consent is not approved,
// and false for non-minor or approved accounts. An error is returned only for real DB failures.
func IsCoppaMinorBlocked(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	var isMinor bool
	var status string
	err := pool.QueryRow(ctx, `
SELECT coppa_minor, coppa_consent_status
  FROM "user".users
 WHERE id = $1`, userID).Scan(&isMinor, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return isMinor && ConsentStatus(status) != ConsentStatusApproved, nil
}

// InitiateEmailConsent creates a pending consent record and returns a raw token to be emailed to the parent.
// The raw token is never stored — only its SHA-256 hash persists in the DB (16 CFR §312.7).
func InitiateEmailConsent(ctx context.Context, pool *pgxpool.Pool, orgID, studentID uuid.UUID, parentEmail string) (*ConsentToken, error) {
	raw, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("coppa: generate token: %w", err)
	}
	hash := hashToken(raw)

	var id uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO compliance.coppa_consents
  (org_id, student_id, parent_email, consent_method, consent_token_hash)
VALUES ($1, $2, $3, 'email_signed', $4)
RETURNING id`,
		orgID, studentID, strings.ToLower(strings.TrimSpace(parentEmail)), hash,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("coppa: insert consent record: %w", err)
	}
	return &ConsentToken{RawToken: raw, ConsentID: id}, nil
}

// ConsumeConsentToken validates a raw token and, if valid, marks the consent as approved and
// activates the student account. It is idempotent if already approved.
func ConsumeConsentToken(ctx context.Context, pool *pgxpool.Pool, rawToken string, now time.Time) (*ConsentRecord, error) {
	tok := strings.TrimSpace(rawToken)
	if tok == "" {
		return nil, ErrTokenInvalid
	}
	hash := hashToken(tok)
	cutoff := now.UTC().Add(-consentTokenTTL)

	var rec ConsentRecord
	var consentedAt *time.Time
	var revokedAt *time.Time
	var priorID *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, org_id, student_id, parent_email, consent_method,
       consented_at, revoked_at, prior_record_id, ai_features_enabled, created_at
  FROM compliance.coppa_consents
 WHERE consent_token_hash = $1
 LIMIT 1`, hash).Scan(
		&rec.ID, &rec.OrgID, &rec.StudentID, &rec.ParentEmail, &rec.ConsentMethod,
		&consentedAt, &revokedAt, &priorID, &rec.AIFeaturesEnabled, &rec.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenInvalid
		}
		return nil, err
	}
	rec.ConsentedAt = consentedAt
	rec.RevokedAt = revokedAt
	rec.PriorRecordID = priorID

	if rec.CreatedAt.Before(cutoff) {
		return nil, ErrTokenExpired
	}
	if rec.ConsentedAt != nil {
		return &rec, ErrAlreadyApproved
	}

	// Mark approved and activate student in one transaction.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	t := now.UTC()
	_, err = tx.Exec(ctx, `
UPDATE compliance.coppa_consents
   SET consented_at = $2
 WHERE id = $1`, rec.ID, t)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
UPDATE "user".users
   SET coppa_consent_status = 'approved'
 WHERE id = $1`, rec.StudentID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	rec.ConsentedAt = &t
	return &rec, nil
}

// RevokeConsent inserts a revocation record (amendments are new rows per FR-9) and freezes the account.
func RevokeConsent(ctx context.Context, pool *pgxpool.Pool, consentID, parentUserID uuid.UUID, now time.Time) error {
	var orgID, studentID uuid.UUID
	var parentEmail string
	var revokedAt *time.Time
	var consentedAt *time.Time
	err := pool.QueryRow(ctx, `
SELECT org_id, student_id, parent_email, revoked_at, consented_at
  FROM compliance.coppa_consents
 WHERE id = $1`, consentID).Scan(&orgID, &studentID, &parentEmail, &revokedAt, &consentedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if revokedAt != nil {
		return ErrAlreadyRevoked
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	t := now.UTC()
	// FR-9: amendments create new records referencing prior.
	_, err = tx.Exec(ctx, `
INSERT INTO compliance.coppa_consents
  (org_id, student_id, parent_email, consent_method, revoked_at, prior_record_id)
VALUES ($1, $2, $3, 'direct', $4, $5)`,
		orgID, studentID, parentEmail, t, consentID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
UPDATE "user".users
   SET coppa_consent_status = 'revoked'
 WHERE id = $1`, studentID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SetAIOptIn updates ai_features_enabled on the active consent record for a student.
func SetAIOptIn(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, enabled bool) error {
	tag, err := pool.Exec(ctx, `
UPDATE compliance.coppa_consents
   SET ai_features_enabled = $2
 WHERE student_id = $1
   AND revoked_at IS NULL
   AND consented_at IS NOT NULL`, studentID, enabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// BulkSchoolAuthorization processes a slice of district consent rows, activating each matched student.
func BulkSchoolAuthorization(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, rows []BulkImportRow) BulkImportResult {
	var result BulkImportResult
	now := time.Now().UTC()
	for _, row := range rows {
		method := row.ConsentMethod
		if method == "" {
			method = ConsentMethodSchoolAuthorization
		}
		consentedAt := row.ConsentDate
		if consentedAt.IsZero() {
			consentedAt = now
		}
		_, err := pool.Exec(ctx, `
INSERT INTO compliance.coppa_consents
  (org_id, student_id, parent_email, consent_method, consented_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT DO NOTHING`,
			orgID, row.StudentID, strings.ToLower(strings.TrimSpace(row.ParentEmail)), string(method), consentedAt,
		)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("student %s: %v", row.StudentID, err))
			result.Skipped++
			continue
		}
		_, err = pool.Exec(ctx, `
UPDATE "user".users
   SET coppa_consent_status = 'approved'
 WHERE id = $1 AND coppa_minor = true`, row.StudentID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("student %s activate: %v", row.StudentID, err))
			result.Skipped++
			continue
		}
		result.Imported++
	}
	return result
}

// ListConsentRecords returns all consent records for a student, newest first.
func ListConsentRecords(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]ConsentRecord, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, student_id, parent_email, consent_method,
       consented_at, revoked_at, prior_record_id, ai_features_enabled, created_at
  FROM compliance.coppa_consents
 WHERE student_id = $1
 ORDER BY created_at DESC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ConsentRecord
	for rows.Next() {
		var rec ConsentRecord
		var consentedAt, revokedAt *time.Time
		var priorID *uuid.UUID
		if err := rows.Scan(
			&rec.ID, &rec.OrgID, &rec.StudentID, &rec.ParentEmail, &rec.ConsentMethod,
			&consentedAt, &revokedAt, &priorID, &rec.AIFeaturesEnabled, &rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		rec.ConsentedAt = consentedAt
		rec.RevokedAt = revokedAt
		rec.PriorRecordID = priorID
		records = append(records, rec)
	}
	return records, rows.Err()
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
