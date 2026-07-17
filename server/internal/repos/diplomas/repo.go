// Package diplomas stores diploma/certificate templates and issued credentials (T11).
package diplomas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Kind is diploma or formal certificate.
type Kind string

const (
	KindDiploma     Kind = "diploma"
	KindCertificate Kind = "certificate"
)

// Template is a registrar-designed diploma/certificate layout.
type Template struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	Kind          Kind
	Name          string
	Title         string
	Program       *string
	ConferralText *string
	Layout        json.RawMessage
	Active        bool
	CreatedBy     *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Diploma is an issued, signed credential artifact.
type Diploma struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	OrgID            uuid.UUID
	TemplateID       *uuid.UUID
	Kind             Kind
	CredentialTitle  string
	Program          *string
	Honors           *string
	ConferredAt      time.Time
	Version          int
	ReplacesID       *uuid.UUID
	Canonical        json.RawMessage
	ContentHash      string
	PDFBytes         []byte
	PDFKey           *string
	VCProof          json.RawMessage
	VerifyToken      *string
	RevokedAt        *time.Time
	RevokeReason     *string
	IssuedBy         *uuid.UUID
	IssuedAt         time.Time
	ProgramRef       *uuid.UUID
}

// Batch tracks cohort issuance progress.
type Batch struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	TemplateID   uuid.UUID
	ProgramRef   *uuid.UUID
	Program      *string
	Honors       *string
	ConferredAt  time.Time
	Status       string
	TotalCount   int
	SuccessCount int
	FailCount    int
	SkipCount    int
	ErrorSummary *string
	CreatedBy    *uuid.UUID
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

// BatchItem is one learner row in a batch.
type BatchItem struct {
	ID        uuid.UUID
	BatchID   uuid.UUID
	UserID    uuid.UUID
	DiplomaID *uuid.UUID
	Status    string
	Error     *string
	CreatedAt time.Time
}

// CreateTemplateInput creates a new template.
type CreateTemplateInput struct {
	OrgID         uuid.UUID
	Kind          Kind
	Name          string
	Title         string
	Program       *string
	ConferralText *string
	Layout        json.RawMessage
	CreatedBy     *uuid.UUID
}

// UpdateTemplateInput patches a template.
type UpdateTemplateInput struct {
	Name          *string
	Title         *string
	Program       *string
	ConferralText *string
	Layout        json.RawMessage
	Active        *bool
	ClearProgram  bool
	ClearText     bool
}

// InsertDiplomaInput persists a newly issued credential.
type InsertDiplomaInput struct {
	UserID          uuid.UUID
	OrgID           uuid.UUID
	TemplateID      *uuid.UUID
	Kind            Kind
	CredentialTitle string
	Program         *string
	Honors          *string
	ConferredAt     time.Time
	Version         int
	ReplacesID      *uuid.UUID
	Canonical       json.RawMessage
	ContentHash     string
	PDFBytes        []byte
	PDFKey          *string
	VCProof         json.RawMessage
	VerifyToken     string
	IssuedBy        *uuid.UUID
	ProgramRef      *uuid.UUID
}

var (
	ErrNotFound      = errors.New("diplomas: not found")
	ErrDuplicate     = errors.New("diplomas: already issued for this learner/template/program")
	ErrInvalidKind   = errors.New("diplomas: invalid kind")
	ErrInvalidInput  = errors.New("diplomas: invalid input")
)

func normalizeKind(k Kind) (Kind, error) {
	switch Kind(strings.ToLower(strings.TrimSpace(string(k)))) {
	case KindDiploma:
		return KindDiploma, nil
	case KindCertificate:
		return KindCertificate, nil
	default:
		return "", ErrInvalidKind
	}
}

// CreateTemplate inserts a diploma/certificate template.
func CreateTemplate(ctx context.Context, pool *pgxpool.Pool, in CreateTemplateInput) (*Template, error) {
	kind, err := normalizeKind(in.Kind)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || in.OrgID == uuid.Nil {
		return nil, ErrInvalidInput
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = name
	}
	layout := in.Layout
	if len(layout) == 0 {
		layout = json.RawMessage(`{}`)
	}
	var t Template
	err = pool.QueryRow(ctx, `
INSERT INTO credentials.diploma_templates (
    org_id, kind, name, title, program, conferral_text, layout, created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
RETURNING id, org_id, kind, name, title, program, conferral_text, layout, active, created_by, created_at, updated_at
`, in.OrgID, string(kind), name, title, in.Program, in.ConferralText, layout, in.CreatedBy).Scan(
		&t.ID, &t.OrgID, &t.Kind, &t.Name, &t.Title, &t.Program, &t.ConferralText,
		&t.Layout, &t.Active, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("diplomas: create template: %w", err)
	}
	return &t, nil
}

// UpdateTemplate updates template fields.
func UpdateTemplate(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, in UpdateTemplateInput) (*Template, error) {
	cur, err := GetTemplateByID(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrNotFound
	}
	name := cur.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
	}
	title := cur.Title
	if in.Title != nil {
		title = strings.TrimSpace(*in.Title)
		if title == "" {
			title = name
		}
	}
	program := cur.Program
	if in.ClearProgram {
		program = nil
	} else if in.Program != nil {
		program = in.Program
	}
	conferral := cur.ConferralText
	if in.ClearText {
		conferral = nil
	} else if in.ConferralText != nil {
		conferral = in.ConferralText
	}
	layout := cur.Layout
	if len(in.Layout) > 0 {
		layout = in.Layout
	}
	active := cur.Active
	if in.Active != nil {
		active = *in.Active
	}
	var t Template
	err = pool.QueryRow(ctx, `
UPDATE credentials.diploma_templates
SET name = $2, title = $3, program = $4, conferral_text = $5, layout = $6::jsonb, active = $7, updated_at = NOW()
WHERE id = $1
RETURNING id, org_id, kind, name, title, program, conferral_text, layout, active, created_by, created_at, updated_at
`, id, name, title, program, conferral, layout, active).Scan(
		&t.ID, &t.OrgID, &t.Kind, &t.Name, &t.Title, &t.Program, &t.ConferralText,
		&t.Layout, &t.Active, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("diplomas: update template: %w", err)
	}
	return &t, nil
}

// GetTemplateByID loads one template.
func GetTemplateByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Template, error) {
	var t Template
	err := pool.QueryRow(ctx, `
SELECT id, org_id, kind, name, title, program, conferral_text, layout, active, created_by, created_at, updated_at
FROM credentials.diploma_templates WHERE id = $1
`, id).Scan(
		&t.ID, &t.OrgID, &t.Kind, &t.Name, &t.Title, &t.Program, &t.ConferralText,
		&t.Layout, &t.Active, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("diplomas: get template: %w", err)
	}
	return &t, nil
}

// ListTemplates returns org templates (active first).
func ListTemplates(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, activeOnly bool) ([]Template, error) {
	q := `
SELECT id, org_id, kind, name, title, program, conferral_text, layout, active, created_by, created_at, updated_at
FROM credentials.diploma_templates
WHERE org_id = $1
`
	if activeOnly {
		q += ` AND active = TRUE`
	}
	q += ` ORDER BY active DESC, created_at DESC`
	rows, err := pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("diplomas: list templates: %w", err)
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(
			&t.ID, &t.OrgID, &t.Kind, &t.Name, &t.Title, &t.Program, &t.ConferralText,
			&t.Layout, &t.Active, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// InsertDiploma stores an issued credential (idempotent unique constraint).
func InsertDiploma(ctx context.Context, pool *pgxpool.Pool, in InsertDiplomaInput) (*Diploma, error) {
	kind, err := normalizeKind(in.Kind)
	if err != nil {
		return nil, err
	}
	if in.UserID == uuid.Nil || in.OrgID == uuid.Nil || strings.TrimSpace(in.CredentialTitle) == "" || in.ContentHash == "" {
		return nil, ErrInvalidInput
	}
	version := in.Version
	if version < 1 {
		version = 1
	}
	canonical := in.Canonical
	if len(canonical) == 0 {
		canonical = json.RawMessage(`{}`)
	}
	var d Diploma
	err = pool.QueryRow(ctx, `
INSERT INTO credentials.diplomas (
    user_id, org_id, template_id, kind, credential_title, program, honors, conferred_at,
    version, replaces_id, canonical, content_hash, pdf_bytes, pdf_key, vc_proof, verify_token,
    issued_by, program_ref
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11::jsonb, $12, $13, $14, $15::jsonb, NULLIF($16, ''),
    $17, $18
)
RETURNING id, user_id, org_id, template_id, kind, credential_title, program, honors, conferred_at,
          version, replaces_id, canonical, content_hash, pdf_bytes, pdf_key, vc_proof, verify_token,
          revoked_at, revoke_reason, issued_by, issued_at, program_ref
`, in.UserID, in.OrgID, in.TemplateID, string(kind), strings.TrimSpace(in.CredentialTitle),
		in.Program, in.Honors, in.ConferredAt.UTC(), version, in.ReplacesID, canonical, in.ContentHash,
		in.PDFBytes, in.PDFKey, in.VCProof, strings.TrimSpace(in.VerifyToken), in.IssuedBy, in.ProgramRef,
	).Scan(
		&d.ID, &d.UserID, &d.OrgID, &d.TemplateID, &d.Kind, &d.CredentialTitle, &d.Program, &d.Honors,
		&d.ConferredAt, &d.Version, &d.ReplacesID, &d.Canonical, &d.ContentHash, &d.PDFBytes, &d.PDFKey,
		&d.VCProof, &d.VerifyToken, &d.RevokedAt, &d.RevokeReason, &d.IssuedBy, &d.IssuedAt, &d.ProgramRef,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("diplomas: insert: %w", err)
	}
	return &d, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(strings.ToLower(msg), "unique")
}

// FindExisting returns an active (or any) diploma for the idempotency key.
func FindExisting(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, templateID *uuid.UUID, programRef *uuid.UUID) (*Diploma, error) {
	if userID == uuid.Nil {
		return nil, nil
	}
	var d Diploma
	err := pool.QueryRow(ctx, `
SELECT id, user_id, org_id, template_id, kind, credential_title, program, honors, conferred_at,
       version, replaces_id, canonical, content_hash, pdf_bytes, pdf_key, vc_proof, verify_token,
       revoked_at, revoke_reason, issued_by, issued_at, program_ref
FROM credentials.diplomas
WHERE user_id = $1
  AND template_id IS NOT DISTINCT FROM $2
  AND program_ref IS NOT DISTINCT FROM $3
ORDER BY version DESC
LIMIT 1
`, userID, templateID, programRef).Scan(
		&d.ID, &d.UserID, &d.OrgID, &d.TemplateID, &d.Kind, &d.CredentialTitle, &d.Program, &d.Honors,
		&d.ConferredAt, &d.Version, &d.ReplacesID, &d.Canonical, &d.ContentHash, &d.PDFBytes, &d.PDFKey,
		&d.VCProof, &d.VerifyToken, &d.RevokedAt, &d.RevokeReason, &d.IssuedBy, &d.IssuedAt, &d.ProgramRef,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("diplomas: find existing: %w", err)
	}
	return &d, nil
}

// GetByID loads a diploma by id.
func GetByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Diploma, error) {
	var d Diploma
	err := pool.QueryRow(ctx, `
SELECT id, user_id, org_id, template_id, kind, credential_title, program, honors, conferred_at,
       version, replaces_id, canonical, content_hash, pdf_bytes, pdf_key, vc_proof, verify_token,
       revoked_at, revoke_reason, issued_by, issued_at, program_ref
FROM credentials.diplomas WHERE id = $1
`, id).Scan(
		&d.ID, &d.UserID, &d.OrgID, &d.TemplateID, &d.Kind, &d.CredentialTitle, &d.Program, &d.Honors,
		&d.ConferredAt, &d.Version, &d.ReplacesID, &d.Canonical, &d.ContentHash, &d.PDFBytes, &d.PDFKey,
		&d.VCProof, &d.VerifyToken, &d.RevokedAt, &d.RevokeReason, &d.IssuedBy, &d.IssuedAt, &d.ProgramRef,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("diplomas: get: %w", err)
	}
	return &d, nil
}

// GetByVerifyToken resolves a public verify token.
func GetByVerifyToken(ctx context.Context, pool *pgxpool.Pool, token string) (*Diploma, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	var d Diploma
	err := pool.QueryRow(ctx, `
SELECT id, user_id, org_id, template_id, kind, credential_title, program, honors, conferred_at,
       version, replaces_id, canonical, content_hash, pdf_bytes, pdf_key, vc_proof, verify_token,
       revoked_at, revoke_reason, issued_by, issued_at, program_ref
FROM credentials.diplomas WHERE verify_token = $1
`, token).Scan(
		&d.ID, &d.UserID, &d.OrgID, &d.TemplateID, &d.Kind, &d.CredentialTitle, &d.Program, &d.Honors,
		&d.ConferredAt, &d.Version, &d.ReplacesID, &d.Canonical, &d.ContentHash, &d.PDFBytes, &d.PDFKey,
		&d.VCProof, &d.VerifyToken, &d.RevokedAt, &d.RevokeReason, &d.IssuedBy, &d.IssuedAt, &d.ProgramRef,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("diplomas: get by token: %w", err)
	}
	return &d, nil
}

// ListByUser returns issued credentials for a learner.
func ListByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Diploma, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, org_id, template_id, kind, credential_title, program, honors, conferred_at,
       version, replaces_id, canonical, content_hash, pdf_bytes, pdf_key, vc_proof, verify_token,
       revoked_at, revoke_reason, issued_by, issued_at, program_ref
FROM credentials.diplomas
WHERE user_id = $1
ORDER BY issued_at DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("diplomas: list by user: %w", err)
	}
	defer rows.Close()
	return scanDiplomas(rows)
}

func scanDiplomas(rows pgx.Rows) ([]Diploma, error) {
	var out []Diploma
	for rows.Next() {
		var d Diploma
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.OrgID, &d.TemplateID, &d.Kind, &d.CredentialTitle, &d.Program, &d.Honors,
			&d.ConferredAt, &d.Version, &d.ReplacesID, &d.Canonical, &d.ContentHash, &d.PDFBytes, &d.PDFKey,
			&d.VCProof, &d.VerifyToken, &d.RevokedAt, &d.RevokeReason, &d.IssuedBy, &d.IssuedAt, &d.ProgramRef,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Revoke marks a diploma revoked.
func Revoke(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, reason string) (*Diploma, error) {
	reason = strings.TrimSpace(reason)
	var d Diploma
	err := pool.QueryRow(ctx, `
UPDATE credentials.diplomas
SET revoked_at = COALESCE(revoked_at, NOW()), revoke_reason = NULLIF($2, '')
WHERE id = $1
RETURNING id, user_id, org_id, template_id, kind, credential_title, program, honors, conferred_at,
          version, replaces_id, canonical, content_hash, pdf_bytes, pdf_key, vc_proof, verify_token,
          revoked_at, revoke_reason, issued_by, issued_at, program_ref
`, id, reason).Scan(
		&d.ID, &d.UserID, &d.OrgID, &d.TemplateID, &d.Kind, &d.CredentialTitle, &d.Program, &d.Honors,
		&d.ConferredAt, &d.Version, &d.ReplacesID, &d.Canonical, &d.ContentHash, &d.PDFBytes, &d.PDFKey,
		&d.VCProof, &d.VerifyToken, &d.RevokedAt, &d.RevokeReason, &d.IssuedBy, &d.IssuedAt, &d.ProgramRef,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("diplomas: revoke: %w", err)
	}
	return &d, nil
}

// Unrevoke clears revocation.
func Unrevoke(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Diploma, error) {
	var d Diploma
	err := pool.QueryRow(ctx, `
UPDATE credentials.diplomas
SET revoked_at = NULL, revoke_reason = NULL
WHERE id = $1
RETURNING id, user_id, org_id, template_id, kind, credential_title, program, honors, conferred_at,
          version, replaces_id, canonical, content_hash, pdf_bytes, pdf_key, vc_proof, verify_token,
          revoked_at, revoke_reason, issued_by, issued_at, program_ref
`, id).Scan(
		&d.ID, &d.UserID, &d.OrgID, &d.TemplateID, &d.Kind, &d.CredentialTitle, &d.Program, &d.Honors,
		&d.ConferredAt, &d.Version, &d.ReplacesID, &d.Canonical, &d.ContentHash, &d.PDFBytes, &d.PDFKey,
		&d.VCProof, &d.VerifyToken, &d.RevokedAt, &d.RevokeReason, &d.IssuedBy, &d.IssuedAt, &d.ProgramRef,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("diplomas: unrevoke: %w", err)
	}
	return &d, nil
}

// CreateBatch starts a cohort issuance batch with pending items.
func CreateBatch(ctx context.Context, pool *pgxpool.Pool, orgID, templateID uuid.UUID, programRef *uuid.UUID, program, honors *string, conferredAt time.Time, createdBy *uuid.UUID, userIDs []uuid.UUID) (*Batch, error) {
	if orgID == uuid.Nil || templateID == uuid.Nil || len(userIDs) == 0 {
		return nil, ErrInvalidInput
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var b Batch
	err = tx.QueryRow(ctx, `
INSERT INTO credentials.diploma_batches (
    org_id, template_id, program_ref, program, honors, conferred_at, status, total_count, created_by
) VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7, $8)
RETURNING id, org_id, template_id, program_ref, program, honors, conferred_at, status,
          total_count, success_count, fail_count, skip_count, error_summary, created_by, created_at, started_at, finished_at
`, orgID, templateID, programRef, program, honors, conferredAt.UTC(), len(userIDs), createdBy).Scan(
		&b.ID, &b.OrgID, &b.TemplateID, &b.ProgramRef, &b.Program, &b.Honors, &b.ConferredAt, &b.Status,
		&b.TotalCount, &b.SuccessCount, &b.FailCount, &b.SkipCount, &b.ErrorSummary, &b.CreatedBy,
		&b.CreatedAt, &b.StartedAt, &b.FinishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("diplomas: create batch: %w", err)
	}
	for _, uid := range userIDs {
		if uid == uuid.Nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO credentials.diploma_batch_items (batch_id, user_id, status)
VALUES ($1, $2, 'pending')
ON CONFLICT (batch_id, user_id) DO NOTHING
`, b.ID, uid); err != nil {
			return nil, fmt.Errorf("diplomas: batch item: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetBatch loads a batch.
func GetBatch(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Batch, error) {
	var b Batch
	err := pool.QueryRow(ctx, `
SELECT id, org_id, template_id, program_ref, program, honors, conferred_at, status,
       total_count, success_count, fail_count, skip_count, error_summary, created_by, created_at, started_at, finished_at
FROM credentials.diploma_batches WHERE id = $1
`, id).Scan(
		&b.ID, &b.OrgID, &b.TemplateID, &b.ProgramRef, &b.Program, &b.Honors, &b.ConferredAt, &b.Status,
		&b.TotalCount, &b.SuccessCount, &b.FailCount, &b.SkipCount, &b.ErrorSummary, &b.CreatedBy,
		&b.CreatedAt, &b.StartedAt, &b.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("diplomas: get batch: %w", err)
	}
	return &b, nil
}

// ListPendingBatchItems returns items still pending/failed (for resume).
func ListPendingBatchItems(ctx context.Context, pool *pgxpool.Pool, batchID uuid.UUID) ([]BatchItem, error) {
	rows, err := pool.Query(ctx, `
SELECT id, batch_id, user_id, diploma_id, status, error, created_at
FROM credentials.diploma_batch_items
WHERE batch_id = $1 AND status IN ('pending', 'failed')
ORDER BY created_at
`, batchID)
	if err != nil {
		return nil, fmt.Errorf("diplomas: list batch items: %w", err)
	}
	defer rows.Close()
	var out []BatchItem
	for rows.Next() {
		var it BatchItem
		if err := rows.Scan(&it.ID, &it.BatchID, &it.UserID, &it.DiplomaID, &it.Status, &it.Error, &it.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// MarkBatchRunning sets status=running.
func MarkBatchRunning(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE credentials.diploma_batches
SET status = 'running', started_at = COALESCE(started_at, NOW())
WHERE id = $1
`, id)
	return err
}

// UpdateBatchItemResult records one item outcome and increments counters.
func UpdateBatchItemResult(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, status string, diplomaID *uuid.UUID, errMsg string) error {
	_, err := pool.Exec(ctx, `
UPDATE credentials.diploma_batch_items
SET status = $2, diploma_id = $3, error = NULLIF($4, '')
WHERE id = $1
`, itemID, status, diplomaID, strings.TrimSpace(errMsg))
	if err != nil {
		return err
	}
	col := ""
	switch status {
	case "issued":
		col = "success_count"
	case "skipped":
		col = "skip_count"
	case "failed":
		col = "fail_count"
	default:
		return nil
	}
	_, err = pool.Exec(ctx, fmt.Sprintf(`
UPDATE credentials.diploma_batches b
SET %s = %s + 1
FROM credentials.diploma_batch_items i
WHERE i.id = $1 AND b.id = i.batch_id
`, col, col), itemID)
	return err
}

// FinishBatch marks a batch completed/failed.
func FinishBatch(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status, errSummary string) error {
	_, err := pool.Exec(ctx, `
UPDATE credentials.diploma_batches
SET status = $2, error_summary = NULLIF($3, ''), finished_at = NOW()
WHERE id = $1
`, id, status, strings.TrimSpace(errSummary))
	return err
}

// VerifyContentHash checks stored hash matches recomputed SHA-256 of canonical JSON.
func VerifyContentHash(d *Diploma) bool {
	if d == nil || d.ContentHash == "" || len(d.Canonical) == 0 {
		return false
	}
	return d.ContentHash == HashCanonical(d.Canonical)
}

// HashCanonical returns hex SHA-256 of canonical bytes.
func HashCanonical(canonical json.RawMessage) string {
	sum := sha256Sum(canonical)
	return sum
}
