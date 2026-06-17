// Package credentials persists issued Open Badges credentials (plans 15.5, 15.6).
package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SourceType identifies what completion produced the credential.
type SourceType string

const (
	SourceCourse SourceType = "course"
	SourcePath   SourceType = "path"
	SourceCEU    SourceType = "ceu"
)

// IssuedCredential is one row in credentials.issued_credentials.
type IssuedCredential struct {
	ID             uuid.UUID
	RecipientID    uuid.UUID
	TemplateID     *uuid.UUID
	SourceType     SourceType
	SourceID       uuid.UUID
	Title          string
	CredentialJSON json.RawMessage
	Proof          json.RawMessage
	PDFKey         *string
	Revoked        bool
	IssuedAt       time.Time
}

// ListByRecipient returns credentials for a learner ordered by issued_at desc.
func ListByRecipient(ctx context.Context, pool *pgxpool.Pool, recipientID uuid.UUID) ([]IssuedCredential, error) {
	rows, err := pool.Query(ctx, `
SELECT id, recipient_id, template_id, source_type, source_id, title,
       credential_json, proof, pdf_key, revoked, issued_at
FROM credentials.issued_credentials
WHERE recipient_id = $1
ORDER BY issued_at DESC
`, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IssuedCredential
	for rows.Next() {
		var row IssuedCredential
		var sourceType string
		if err := rows.Scan(
			&row.ID, &row.RecipientID, &row.TemplateID, &sourceType, &row.SourceID, &row.Title,
			&row.CredentialJSON, &row.Proof, &row.PDFKey, &row.Revoked, &row.IssuedAt,
		); err != nil {
			return nil, err
		}
		row.SourceType = SourceType(sourceType)
		out = append(out, row)
	}
	return out, rows.Err()
}

// GetByID returns a credential by id, or nil if not found.
func GetByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*IssuedCredential, error) {
	row := IssuedCredential{}
	var sourceType string
	err := pool.QueryRow(ctx, `
SELECT id, recipient_id, template_id, source_type, source_id, title,
       credential_json, proof, pdf_key, revoked, issued_at
FROM credentials.issued_credentials
WHERE id = $1
`, id).Scan(
		&row.ID, &row.RecipientID, &row.TemplateID, &sourceType, &row.SourceID, &row.Title,
		&row.CredentialJSON, &row.Proof, &row.PDFKey, &row.Revoked, &row.IssuedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.SourceType = SourceType(sourceType)
	return &row, nil
}

// GetByRecipientAndSource returns an existing credential for idempotent issuance.
func GetByRecipientAndSource(
	ctx context.Context,
	pool *pgxpool.Pool,
	recipientID uuid.UUID,
	sourceType SourceType,
	sourceID uuid.UUID,
) (*IssuedCredential, error) {
	row := IssuedCredential{}
	var st string
	err := pool.QueryRow(ctx, `
SELECT id, recipient_id, template_id, source_type, source_id, title,
       credential_json, proof, pdf_key, revoked, issued_at
FROM credentials.issued_credentials
WHERE recipient_id = $1 AND source_type = $2 AND source_id = $3
`, recipientID, string(sourceType), sourceID).Scan(
		&row.ID, &row.RecipientID, &row.TemplateID, &st, &row.SourceID, &row.Title,
		&row.CredentialJSON, &row.Proof, &row.PDFKey, &row.Revoked, &row.IssuedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.SourceType = SourceType(st)
	return &row, nil
}

// Create inserts a new issued credential.
func Create(ctx context.Context, pool *pgxpool.Pool, cred IssuedCredential) (*IssuedCredential, error) {
	var sourceType string
	err := pool.QueryRow(ctx, `
INSERT INTO credentials.issued_credentials (
    recipient_id, template_id, source_type, source_id, title,
    credential_json, proof, pdf_key, issued_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, NOW()))
RETURNING id, recipient_id, template_id, source_type, source_id, title,
          credential_json, proof, pdf_key, revoked, issued_at
`, cred.RecipientID, cred.TemplateID, string(cred.SourceType), cred.SourceID, cred.Title,
		cred.CredentialJSON, cred.Proof, cred.PDFKey, nullableTime(cred.IssuedAt),
	).Scan(
		&cred.ID, &cred.RecipientID, &cred.TemplateID, &sourceType, &cred.SourceID, &cred.Title,
		&cred.CredentialJSON, &cred.Proof, &cred.PDFKey, &cred.Revoked, &cred.IssuedAt,
	)
	if err != nil {
		return nil, err
	}
	cred.SourceType = SourceType(sourceType)
	return &cred, nil
}

// CourseMeta is course title metadata for credential issuance.
type CourseMeta struct {
	ID    uuid.UUID
	Title string
}

// CourseMetaByID loads course title for issuance emails and credential naming.
func CourseMetaByID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*CourseMeta, error) {
	var meta CourseMeta
	err := pool.QueryRow(ctx, `
SELECT id, COALESCE(NULLIF(TRIM(title), ''), course_code)
FROM course.courses
WHERE id = $1
`, courseID).Scan(&meta.ID, &meta.Title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}