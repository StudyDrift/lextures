// Package ccr persists co-curricular transcript achievements and generated documents (plan 14.13).
package ccr

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AchievementType identifies how an achievement was sourced.
type AchievementType string

const (
	TypeCourseCompletion AchievementType = "course_completion"
	TypeBadge            AchievementType = "badge"
	TypeCertificate      AchievementType = "certificate"
	TypePortfolio        AchievementType = "portfolio"
	TypeExtracurricular  AchievementType = "extracurricular"
)

// Achievement is one row in user.ccr_achievements.
type Achievement struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	AchievementType AchievementType
	SourceID        *uuid.UUID
	Title           string
	Description     *string
	IssuedAt        time.Time
	EvidenceURL     *string
	OutcomeTags     []string
	AddedBy         *uuid.UUID
	CreatedAt       time.Time
}

// UpsertAchievementParams inserts or updates a derived/manual achievement row.
type UpsertAchievementParams struct {
	UserID          uuid.UUID
	AchievementType AchievementType
	SourceID        *uuid.UUID
	Title           string
	Description     *string
	IssuedAt        time.Time
	EvidenceURL     *string
	OutcomeTags     []string
	AddedBy         *uuid.UUID
}

// Document is one generated CLR stored for a student.
type Document struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	GeneratedAt time.Time
	ConsentedAt *time.Time
	CLRJSON     json.RawMessage
	VCProof     json.RawMessage
	PDFKey      *string
	ShareToken  *string
	CreatedAt   time.Time
}

// InsertDocumentParams stores a newly generated CLR document.
type InsertDocumentParams struct {
	UserID      uuid.UUID
	ConsentedAt *time.Time
	CLRJSON     json.RawMessage
	VCProof     json.RawMessage
	PDFKey      *string
	ShareToken  *string
}

// SigningConfig holds the institutional VC signing key material.
type SigningConfig struct {
	ID               int
	IssuerDID        string
	PublicKeyJWK     json.RawMessage
	PrivateKeyCipher []byte
	UpdatedAt        time.Time
}

// ListAchievementsByUser returns achievements newest first.
func ListAchievementsByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Achievement, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, achievement_type, source_id, title, description, issued_at,
       evidence_url, outcome_tags, added_by, created_at
FROM user.ccr_achievements
WHERE user_id = $1
ORDER BY issued_at DESC, created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Achievement, 0)
	for rows.Next() {
		var a Achievement
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.AchievementType, &a.SourceID, &a.Title, &a.Description,
			&a.IssuedAt, &a.EvidenceURL, &a.OutcomeTags, &a.AddedBy, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpsertAchievement inserts or updates one achievement keyed by user/type/source.
func UpsertAchievement(ctx context.Context, pool *pgxpool.Pool, p UpsertAchievementParams) error {
	_, err := pool.Exec(ctx, `
INSERT INTO user.ccr_achievements
    (user_id, achievement_type, source_id, title, description, issued_at, evidence_url, outcome_tags, added_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, achievement_type, source_id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    issued_at = EXCLUDED.issued_at,
    evidence_url = EXCLUDED.evidence_url,
    outcome_tags = EXCLUDED.outcome_tags,
    added_by = COALESCE(EXCLUDED.added_by, user.ccr_achievements.added_by)
`, p.UserID, p.AchievementType, p.SourceID, p.Title, p.Description, p.IssuedAt, p.EvidenceURL, p.OutcomeTags, p.AddedBy)
	return err
}

// InsertManualAchievement adds an admin-entered extracurricular record.
func InsertManualAchievement(ctx context.Context, pool *pgxpool.Pool, p UpsertAchievementParams) (*Achievement, error) {
	if p.AchievementType != TypeExtracurricular {
		return nil, errors.New("ccr: manual achievements must be extracurricular")
	}
	var a Achievement
	err := pool.QueryRow(ctx, `
INSERT INTO user.ccr_achievements
    (user_id, achievement_type, source_id, title, description, issued_at, evidence_url, outcome_tags, added_by)
VALUES ($1, $2, gen_random_uuid(), $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, achievement_type, source_id, title, description, issued_at,
          evidence_url, outcome_tags, added_by, created_at
`, p.UserID, p.AchievementType, p.Title, p.Description, p.IssuedAt, p.EvidenceURL, p.OutcomeTags, p.AddedBy).Scan(
		&a.ID, &a.UserID, &a.AchievementType, &a.SourceID, &a.Title, &a.Description,
		&a.IssuedAt, &a.EvidenceURL, &a.OutcomeTags, &a.AddedBy, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListDocumentsByUser returns generated documents newest first.
func ListDocumentsByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Document, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, generated_at, consented_at, clr_json, vc_proof, pdf_key, share_token, created_at
FROM user.ccr_documents
WHERE user_id = $1
ORDER BY generated_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Document, 0)
	for rows.Next() {
		var d Document
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.GeneratedAt, &d.ConsentedAt, &d.CLRJSON, &d.VCProof,
			&d.PDFKey, &d.ShareToken, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDocumentByIDForUser loads one document owned by the user.
func GetDocumentByIDForUser(ctx context.Context, pool *pgxpool.Pool, userID, docID uuid.UUID) (*Document, error) {
	var d Document
	err := pool.QueryRow(ctx, `
SELECT id, user_id, generated_at, consented_at, clr_json, vc_proof, pdf_key, share_token, created_at
FROM user.ccr_documents
WHERE id = $1 AND user_id = $2
`, docID, userID).Scan(
		&d.ID, &d.UserID, &d.GeneratedAt, &d.ConsentedAt, &d.CLRJSON, &d.VCProof,
		&d.PDFKey, &d.ShareToken, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDocumentByShareToken loads a publicly shareable document.
func GetDocumentByShareToken(ctx context.Context, pool *pgxpool.Pool, token string) (*Document, error) {
	var d Document
	err := pool.QueryRow(ctx, `
SELECT id, user_id, generated_at, consented_at, clr_json, vc_proof, pdf_key, share_token, created_at
FROM user.ccr_documents
WHERE share_token = $1 AND consented_at IS NOT NULL
`, token).Scan(
		&d.ID, &d.UserID, &d.GeneratedAt, &d.ConsentedAt, &d.CLRJSON, &d.VCProof,
		&d.PDFKey, &d.ShareToken, &d.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// InsertDocument stores a generated CLR document.
func InsertDocument(ctx context.Context, pool *pgxpool.Pool, p InsertDocumentParams) (*Document, error) {
	var d Document
	err := pool.QueryRow(ctx, `
INSERT INTO user.ccr_documents
    (user_id, consented_at, clr_json, vc_proof, pdf_key, share_token)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, generated_at, consented_at, clr_json, vc_proof, pdf_key, share_token, created_at
`, p.UserID, p.ConsentedAt, p.CLRJSON, p.VCProof, p.PDFKey, p.ShareToken).Scan(
		&d.ID, &d.UserID, &d.GeneratedAt, &d.ConsentedAt, &d.CLRJSON, &d.VCProof,
		&d.PDFKey, &d.ShareToken, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetSigningConfig returns the institutional signing key row, if configured.
func GetSigningConfig(ctx context.Context, pool *pgxpool.Pool) (*SigningConfig, error) {
	var c SigningConfig
	err := pool.QueryRow(ctx, `
SELECT id, issuer_did, public_key_jwk, private_key_cipher, updated_at
FROM settings.ccr_signing_config
ORDER BY id ASC
LIMIT 1
`).Scan(&c.ID, &c.IssuerDID, &c.PublicKeyJWK, &c.PrivateKeyCipher, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpsertSigningConfig stores or replaces the institutional signing key row.
func UpsertSigningConfig(ctx context.Context, pool *pgxpool.Pool, issuerDID string, publicJWK json.RawMessage, privateCipher []byte) (*SigningConfig, error) {
	if _, err := pool.Exec(ctx, `DELETE FROM settings.ccr_signing_config`); err != nil {
		return nil, err
	}
	var c SigningConfig
	err := pool.QueryRow(ctx, `
INSERT INTO settings.ccr_signing_config (issuer_did, public_key_jwk, private_key_cipher)
VALUES ($1, $2, $3)
RETURNING id, issuer_did, public_key_jwk, private_key_cipher, updated_at
`, issuerDID, publicJWK, privateCipher).Scan(&c.ID, &c.IssuerDID, &c.PublicKeyJWK, &c.PrivateKeyCipher, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CourseCompletionRow is a derived course completion achievement source.
type CourseCompletionRow struct {
	SubmissionID uuid.UUID
	CourseID     uuid.UUID
	CourseCode   string
	CourseTitle  string
	FinalGrade   string
	SubmittedAt  time.Time
}

// ListCourseCompletions returns final grade submissions for a student.
func ListCourseCompletions(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]CourseCompletionRow, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ON (fgs.enrollment_id)
    fgs.id, fgs.course_id, c.course_code, c.title, fgs.final_grade, fgs.submitted_at
FROM course.final_grade_submissions fgs
JOIN course.course_enrollments e ON e.id = fgs.enrollment_id
JOIN course.courses c ON c.id = fgs.course_id
WHERE e.user_id = $1 AND e.role = 'student'
ORDER BY fgs.enrollment_id, fgs.submitted_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CourseCompletionRow, 0)
	for rows.Next() {
		var r CourseCompletionRow
		if err := rows.Scan(&r.SubmissionID, &r.CourseID, &r.CourseCode, &r.CourseTitle, &r.FinalGrade, &r.SubmittedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
