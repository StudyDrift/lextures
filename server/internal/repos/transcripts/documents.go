package transcripts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DocumentVariant mirrors academicrecord.Variant values stored in the DB.
type DocumentVariant string

const (
	DocumentOfficial   DocumentVariant = "official"
	DocumentUnofficial DocumentVariant = "unofficial"
	DocumentPartial    DocumentVariant = "partial"
	DocumentInProgress DocumentVariant = "in_progress"
)

// Document is an immutable issued transcript artifact.
type Document struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	OrgID            *uuid.UUID
	Variant          DocumentVariant
	Version          int
	Canonical        json.RawMessage
	SchemaVersion    string
	TemplateVersion  string
	ContentHash      string
	PDFBytes         []byte
	PESCXMLBytes     []byte
	PDFKey           *string
	PESCXMLKey       *string
	VCProof          json.RawMessage
	GPACumulative    *float64
	CreditsEarned    *float64
	GeneratedBy      *uuid.UUID
	GeneratedAt      time.Time
	VerifyToken      *string
	PDFHash          *string
	RevokedAt        *time.Time
	RevokeReason     *string
	DisclosePublicly bool
}

// InsertDocumentInput is the payload for creating a new issued document.
type InsertDocumentInput struct {
	UserID           uuid.UUID
	OrgID            *uuid.UUID
	Variant          DocumentVariant
	Canonical        json.RawMessage
	SchemaVersion    string
	TemplateVersion  string
	ContentHash      string
	PDFBytes         []byte
	PESCXMLBytes     []byte
	PDFKey           *string
	PESCXMLKey       *string
	VCProof          json.RawMessage
	GPACumulative    *float64
	CreditsEarned    *float64
	GeneratedBy      *uuid.UUID
	VerifyToken      *string
	PDFHash          *string
	DisclosePublicly bool
}

const documentSelectColumns = `
id, user_id, org_id, variant, version, canonical, schema_version, template_version,
content_hash, pdf_bytes, pesc_xml_bytes, pdf_key, pesc_xml_key, vc_proof,
gpa_cumulative, credits_earned, generated_by, generated_at,
verify_token, pdf_hash, revoked_at, revoke_reason, disclose_publicly`

func scanDocument(row pgx.Row, d *Document) error {
	var variant string
	err := row.Scan(
		&d.ID, &d.UserID, &d.OrgID, &variant, &d.Version, &d.Canonical,
		&d.SchemaVersion, &d.TemplateVersion, &d.ContentHash,
		&d.PDFBytes, &d.PESCXMLBytes, &d.PDFKey, &d.PESCXMLKey, &d.VCProof,
		&d.GPACumulative, &d.CreditsEarned, &d.GeneratedBy, &d.GeneratedAt,
		&d.VerifyToken, &d.PDFHash, &d.RevokedAt, &d.RevokeReason, &d.DisclosePublicly,
	)
	if err != nil {
		return err
	}
	d.Variant = DocumentVariant(variant)
	return nil
}

// NextOfficialVersion returns the next monotonic official version for a user.
func NextOfficialVersion(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int, error) {
	var max *int
	err := pool.QueryRow(ctx, `
SELECT MAX(version) FROM transcripts.transcript_documents
WHERE user_id = $1 AND variant = 'official'
`, userID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if max == nil {
		return 1, nil
	}
	return *max + 1, nil
}

// InsertDocument persists a new immutable transcript document.
// Official variants allocate the next monotonic version; other variants use version 1.
func InsertDocument(ctx context.Context, pool *pgxpool.Pool, in InsertDocumentInput) (*Document, error) {
	version := 1
	if in.Variant == DocumentOfficial {
		v, err := NextOfficialVersion(ctx, pool, in.UserID)
		if err != nil {
			return nil, err
		}
		version = v
	}
	var d Document
	row := pool.QueryRow(ctx, `
INSERT INTO transcripts.transcript_documents (
    user_id, org_id, variant, version, canonical, schema_version, template_version,
    content_hash, pdf_bytes, pesc_xml_bytes, pdf_key, pesc_xml_key, vc_proof,
    gpa_cumulative, credits_earned, generated_by,
    verify_token, pdf_hash, disclose_publicly
)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
RETURNING `+documentSelectColumns+`
`, in.UserID, in.OrgID, string(in.Variant), version, in.Canonical, in.SchemaVersion, in.TemplateVersion,
		in.ContentHash, in.PDFBytes, in.PESCXMLBytes, in.PDFKey, in.PESCXMLKey, in.VCProof,
		in.GPACumulative, in.CreditsEarned, in.GeneratedBy,
		in.VerifyToken, in.PDFHash, in.DisclosePublicly)
	if err := scanDocument(row, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDocumentByID returns a document if it belongs to userID (or admin bypass via org list).
func GetDocumentByID(ctx context.Context, pool *pgxpool.Pool, userID, docID uuid.UUID) (*Document, error) {
	var d Document
	row := pool.QueryRow(ctx, `
SELECT `+documentSelectColumns+`
FROM transcripts.transcript_documents
WHERE id = $1 AND user_id = $2
`, docID, userID)
	if err := scanDocument(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// GetDocumentByIDAdmin returns a document by id without user scoping.
func GetDocumentByIDAdmin(ctx context.Context, pool *pgxpool.Pool, docID uuid.UUID) (*Document, error) {
	var d Document
	row := pool.QueryRow(ctx, `
SELECT `+documentSelectColumns+`
FROM transcripts.transcript_documents
WHERE id = $1
`, docID)
	if err := scanDocument(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// ListDocumentsByUser returns issued documents for a user, newest first.
func ListDocumentsByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Document, error) {
	rows, err := pool.Query(ctx, `
SELECT `+documentSelectColumns+`
FROM transcripts.transcript_documents
WHERE user_id = $1
ORDER BY generated_at DESC
LIMIT 100
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Document
	for rows.Next() {
		var d Document
		if err := scanDocument(rows, &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListDocumentsByStudentAdmin returns documents for a student (registrar view).
func ListDocumentsByStudentAdmin(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]Document, error) {
	return ListDocumentsByUser(ctx, pool, studentID)
}

// VerifyDocumentHash recomputes the SHA-256 of stored canonical JSON and compares to content_hash.
// Returns false (fail closed) when the hash does not match.
func VerifyDocumentHash(d *Document) bool {
	if d == nil || len(d.Canonical) == 0 || d.ContentHash == "" {
		return false
	}
	sum := sha256.Sum256(d.Canonical)
	return hex.EncodeToString(sum[:]) == d.ContentHash
}
