package transcripts

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Verification document types (T08).
const (
	VerifyDocTranscript = "transcript"
	VerifyDocCLR        = "clr"
	VerifyDocDiploma    = "diploma"
)

// Verification results (T08).
const (
	VerifyResultGenuine  = "genuine"
	VerifyResultTampered = "tampered"
	VerifyResultRevoked  = "revoked"
	VerifyResultNotFound = "not_found"
)

// Verification methods (T08).
const (
	VerifyMethodLink   = "link"
	VerifyMethodQR     = "qr"
	VerifyMethodUpload = "upload"
)

// Verification is an audit row for a third-party verify lookup.
type Verification struct {
	ID           uuid.UUID
	DocumentID   *uuid.UUID
	DocumentType string
	Result       string
	Method       string
	RequesterUA  *string
	CreatedAt    time.Time
}

// InsertVerificationInput is the payload for logging a verification attempt.
type InsertVerificationInput struct {
	DocumentID   *uuid.UUID
	DocumentType string
	Result       string
	Method       string
	RequesterIP  *string
	RequesterUA  *string
}

// InsertVerification records a verification attempt.
func InsertVerification(ctx context.Context, pool *pgxpool.Pool, in InsertVerificationInput) (*Verification, error) {
	var v Verification
	row := pool.QueryRow(ctx, `
INSERT INTO transcripts.verifications (document_id, document_type, result, method, requester_ip, requester_ua)
VALUES ($1, $2, $3, $4, $5::inet, $6)
RETURNING id, document_id, document_type, result, method, requester_ua, created_at
`, in.DocumentID, in.DocumentType, in.Result, in.Method, in.RequesterIP, in.RequesterUA)
	if err := row.Scan(&v.ID, &v.DocumentID, &v.DocumentType, &v.Result, &v.Method, &v.RequesterUA, &v.CreatedAt); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetDocumentByVerifyToken loads a transcript document by its public verify token.
func GetDocumentByVerifyToken(ctx context.Context, pool *pgxpool.Pool, token string) (*Document, error) {
	var d Document
	row := pool.QueryRow(ctx, `
SELECT `+documentSelectColumns+`
FROM transcripts.transcript_documents
WHERE verify_token = $1
`, token)
	if err := scanDocument(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// GetDocumentByPDFHash loads a transcript document by SHA-256 of its PDF bytes.
func GetDocumentByPDFHash(ctx context.Context, pool *pgxpool.Pool, pdfHash string) (*Document, error) {
	var d Document
	row := pool.QueryRow(ctx, `
SELECT `+documentSelectColumns+`
FROM transcripts.transcript_documents
WHERE pdf_hash = $1
ORDER BY generated_at DESC
LIMIT 1
`, pdfHash)
	if err := scanDocument(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// RevokeDocument marks an issued transcript as revoked.
func RevokeDocument(ctx context.Context, pool *pgxpool.Pool, docID uuid.UUID, reason string) (*Document, error) {
	var d Document
	row := pool.QueryRow(ctx, `
UPDATE transcripts.transcript_documents
SET revoked_at = NOW(), revoke_reason = NULLIF(TRIM($2), '')
WHERE id = $1
RETURNING `+documentSelectColumns+`
`, docID, reason)
	if err := scanDocument(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// UnrevokeDocument clears revocation on an issued transcript.
func UnrevokeDocument(ctx context.Context, pool *pgxpool.Pool, docID uuid.UUID) (*Document, error) {
	var d Document
	row := pool.QueryRow(ctx, `
UPDATE transcripts.transcript_documents
SET revoked_at = NULL, revoke_reason = NULL
WHERE id = $1
RETURNING `+documentSelectColumns+`
`, docID)
	if err := scanDocument(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// SetDocumentDisclosePublicly updates holder-controlled public disclosure for verify results.
func SetDocumentDisclosePublicly(ctx context.Context, pool *pgxpool.Pool, userID, docID uuid.UUID, disclose bool) (*Document, error) {
	var d Document
	row := pool.QueryRow(ctx, `
UPDATE transcripts.transcript_documents
SET disclose_publicly = $3
WHERE id = $1 AND user_id = $2
RETURNING `+documentSelectColumns+`
`, docID, userID, disclose)
	if err := scanDocument(row, &d); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}
