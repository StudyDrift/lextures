// Package ccr persists co-curricular transcript achievements and generated documents (plan 14.13).
package ccr

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AchievementType is a verified achievement category on the CLR.
type AchievementType string

const (
	TypeCourseCompletion AchievementType = "course_completion"
	TypeBadge            AchievementType = "badge"
	TypeCertificate      AchievementType = "certificate"
	TypePortfolio        AchievementType = "portfolio"
	TypeExtracurricular  AchievementType = "extracurricular"
)

// Achievement is one row in ccr.achievements.
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

// Document is one generated CLR stored in ccr.documents.
type Document struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	GeneratedAt time.Time
	CLRJSON     json.RawMessage
	VCProof     json.RawMessage
	PDFKey      *string
	ShareToken  *string
	CreatedAt   time.Time
}

// CourseCompletion is a derived course completion achievement from final grade submissions.
type CourseCompletion struct {
	CourseID    uuid.UUID
	CourseCode  string
	CourseTitle string
	FinalGrade  string
	IssuedAt    time.Time
}

// ListAchievements returns stored achievements for a user ordered by issued_at desc.
func ListAchievements(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Achievement, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, achievement_type, source_id, title, description, issued_at,
       evidence_url, outcome_tags, added_by, created_at
FROM ccr.achievements
WHERE user_id = $1
ORDER BY issued_at DESC, created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAchievements(rows)
}

// CreateAchievement inserts a manual or synced achievement row.
func CreateAchievement(ctx context.Context, pool *pgxpool.Pool, a Achievement) (*Achievement, error) {
	var out Achievement
	err := pool.QueryRow(ctx, `
INSERT INTO ccr.achievements
    (user_id, achievement_type, source_id, title, description, issued_at, evidence_url, outcome_tags, added_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, user_id, achievement_type, source_id, title, description, issued_at,
          evidence_url, outcome_tags, added_by, created_at
`, a.UserID, string(a.AchievementType), a.SourceID, a.Title, a.Description, a.IssuedAt,
		a.EvidenceURL, a.OutcomeTags, a.AddedBy,
	).Scan(
		&out.ID, &out.UserID, &out.AchievementType, &out.SourceID, &out.Title, &out.Description,
		&out.IssuedAt, &out.EvidenceURL, &out.OutcomeTags, &out.AddedBy, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListCourseCompletions returns latest final grade per enrollment for a student.
func ListCourseCompletions(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]CourseCompletion, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ON (ce.id)
       c.id, c.code, c.title, fgs.final_grade, fgs.submitted_at
FROM course.final_grade_submissions fgs
JOIN course.course_enrollments ce ON ce.id = fgs.enrollment_id
JOIN course.courses c ON c.id = fgs.course_id
WHERE ce.user_id = $1
  AND fgs.final_grade NOT IN ('W', 'I', 'AU', 'NC')
ORDER BY ce.id, fgs.submitted_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CourseCompletion
	for rows.Next() {
		var row CourseCompletion
		if err := rows.Scan(&row.CourseID, &row.CourseCode, &row.CourseTitle, &row.FinalGrade, &row.IssuedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CreateDocument stores a generated CLR document.
func CreateDocument(ctx context.Context, pool *pgxpool.Pool, d Document) (*Document, error) {
	var out Document
	err := pool.QueryRow(ctx, `
INSERT INTO ccr.documents (user_id, clr_json, vc_proof, pdf_key, share_token)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, generated_at, clr_json, vc_proof, pdf_key, share_token, created_at
`, d.UserID, d.CLRJSON, d.VCProof, d.PDFKey, d.ShareToken,
	).Scan(
		&out.ID, &out.UserID, &out.GeneratedAt, &out.CLRJSON, &out.VCProof,
		&out.PDFKey, &out.ShareToken, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListDocuments returns generated documents for a user.
func ListDocuments(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Document, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, generated_at, clr_json, vc_proof, pdf_key, share_token, created_at
FROM ccr.documents
WHERE user_id = $1
ORDER BY generated_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.GeneratedAt, &d.CLRJSON, &d.VCProof,
			&d.PDFKey, &d.ShareToken, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDocumentByID loads a document owned by userID.
func GetDocumentByID(ctx context.Context, pool *pgxpool.Pool, userID, docID uuid.UUID) (*Document, error) {
	var d Document
	err := pool.QueryRow(ctx, `
SELECT id, user_id, generated_at, clr_json, vc_proof, pdf_key, share_token, created_at
FROM ccr.documents
WHERE id = $1 AND user_id = $2
`, docID, userID).Scan(
		&d.ID, &d.UserID, &d.GeneratedAt, &d.CLRJSON, &d.VCProof,
		&d.PDFKey, &d.ShareToken, &d.CreatedAt,
	)
	if err == pgx.ErrNoRows {
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
SELECT id, user_id, generated_at, clr_json, vc_proof, pdf_key, share_token, created_at
FROM ccr.documents
WHERE share_token = $1
`, token).Scan(
		&d.ID, &d.UserID, &d.GeneratedAt, &d.CLRJSON, &d.VCProof,
		&d.PDFKey, &d.ShareToken, &d.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func scanAchievements(rows pgx.Rows) ([]Achievement, error) {
	var out []Achievement
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
