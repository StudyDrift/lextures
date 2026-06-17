// Package credentials persists issued completion credentials (plan 15.5).
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

// SourceType identifies what completion triggered issuance.
type SourceType string

const (
	SourceCourse SourceType = "course"
	SourcePath   SourceType = "path"
	SourceCEU    SourceType = "ceu"
)

// Template is an optional branded certificate template.
type Template struct {
	ID            uuid.UUID
	CourseID      *uuid.UUID
	PathID        *uuid.UUID
	Name          string
	Description   *string
	BackgroundURL *string
	LogoURL       *string
	CreatedAt     time.Time
}

// IssuedCredential is one signed credential row.
type IssuedCredential struct {
	ID             uuid.UUID
	RecipientID    uuid.UUID
	TemplateID     *uuid.UUID
	SourceType     SourceType
	SourceID       uuid.UUID
	CredentialJSON json.RawMessage
	Proof          json.RawMessage
	PDFKey         *string
	Revoked        bool
	IssuedAt       time.Time
}

// ListItem is a summary row for the learner credentials list.
type ListItem struct {
	ID           uuid.UUID
	SourceType   SourceType
	SourceID     uuid.UUID
	Title        string
	IssuedAt     time.Time
	Revoked      bool
	HasPDF       bool
}

// GetByID returns a credential by primary key.
func GetByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*IssuedCredential, error) {
	var out IssuedCredential
	var st string
	err := pool.QueryRow(ctx, `
SELECT id, recipient_id, template_id, source_type, source_id, credential_json, proof, pdf_key, revoked, issued_at
FROM credentials.issued_credentials
WHERE id = $1
`, id).Scan(
		&out.ID, &out.RecipientID, &out.TemplateID, &st, &out.SourceID,
		&out.CredentialJSON, &out.Proof, &out.PDFKey, &out.Revoked, &out.IssuedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out.SourceType = SourceType(st)
	return &out, nil
}

// GetByRecipientSource returns an existing credential for idempotency checks.
func GetByRecipientSource(ctx context.Context, pool *pgxpool.Pool, recipientID uuid.UUID, sourceType SourceType, sourceID uuid.UUID) (*IssuedCredential, error) {
	var out IssuedCredential
	var st string
	err := pool.QueryRow(ctx, `
SELECT id, recipient_id, template_id, source_type, source_id, credential_json, proof, pdf_key, revoked, issued_at
FROM credentials.issued_credentials
WHERE recipient_id = $1 AND source_type = $2 AND source_id = $3
`, recipientID, string(sourceType), sourceID).Scan(
		&out.ID, &out.RecipientID, &out.TemplateID, &st, &out.SourceID,
		&out.CredentialJSON, &out.Proof, &out.PDFKey, &out.Revoked, &out.IssuedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out.SourceType = SourceType(st)
	return &out, nil
}

// InsertIssued stores a new credential; returns false when a duplicate already exists.
func InsertIssued(ctx context.Context, pool *pgxpool.Pool, row IssuedCredential) (*IssuedCredential, bool, error) {
	var out IssuedCredential
	var st string
	err := pool.QueryRow(ctx, `
INSERT INTO credentials.issued_credentials
    (recipient_id, template_id, source_type, source_id, credential_json, proof, pdf_key)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (recipient_id, source_type, source_id) DO NOTHING
RETURNING id, recipient_id, template_id, source_type, source_id, credential_json, proof, pdf_key, revoked, issued_at
`, row.RecipientID, row.TemplateID, string(row.SourceType), row.SourceID, row.CredentialJSON, row.Proof, row.PDFKey).Scan(
		&out.ID, &out.RecipientID, &out.TemplateID, &st, &out.SourceID,
		&out.CredentialJSON, &out.Proof, &out.PDFKey, &out.Revoked, &out.IssuedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, err := GetByRecipientSource(ctx, pool, row.RecipientID, row.SourceType, row.SourceID)
		if err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	out.SourceType = SourceType(st)
	return &out, true, nil
}

// UpdatePDFKey stores the object-storage key after PDF rendering.
func UpdatePDFKey(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, pdfKey string) error {
	_, err := pool.Exec(ctx, `
UPDATE credentials.issued_credentials SET pdf_key = $2 WHERE id = $1
`, id, pdfKey)
	return err
}

// ListForRecipient returns credentials for the learner dashboard.
func ListForRecipient(ctx context.Context, pool *pgxpool.Pool, recipientID uuid.UUID) ([]ListItem, error) {
	rows, err := pool.Query(ctx, `
SELECT ic.id, ic.source_type, ic.source_id, ic.issued_at, ic.revoked, ic.pdf_key IS NOT NULL,
       COALESCE(
         ic.credential_json->'credentialSubject'->'achievement'->>'name',
         ic.credential_json->'name',
         'Certificate'
       ) AS title
FROM credentials.issued_credentials ic
WHERE ic.recipient_id = $1
ORDER BY ic.issued_at DESC
`, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ListItem
	for rows.Next() {
		var item ListItem
		var st string
		if err := rows.Scan(&item.ID, &st, &item.SourceID, &item.IssuedAt, &item.Revoked, &item.HasPDF, &item.Title); err != nil {
			return nil, err
		}
		item.SourceType = SourceType(st)
		out = append(out, item)
	}
	return out, rows.Err()
}

// GetTemplateForCourse returns a course-specific template if configured.
func GetTemplateForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*Template, error) {
	return getTemplate(ctx, pool, `course_id = $1`, courseID)
}

// GetTemplateForPath returns a path-specific template if configured.
func GetTemplateForPath(ctx context.Context, pool *pgxpool.Pool, pathID uuid.UUID) (*Template, error) {
	return getTemplate(ctx, pool, `path_id = $1`, pathID)
}

func getTemplate(ctx context.Context, pool *pgxpool.Pool, where string, id uuid.UUID) (*Template, error) {
	var t Template
	err := pool.QueryRow(ctx, `
SELECT id, course_id, path_id, name, description, background_url, logo_url, created_at
FROM credentials.credential_templates
WHERE `+where+`
`, id).Scan(
		&t.ID, &t.CourseID, &t.PathID, &t.Name, &t.Description, &t.BackgroundURL, &t.LogoURL, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}