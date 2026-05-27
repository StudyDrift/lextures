// Package iso persists ISO 27001/27701 ISMS program data (plan 10.10).
package iso

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditFinding is one row from compliance.iso_audit_findings.
type AuditFinding struct {
	ID               uuid.UUID
	AuditCycle       string
	FindingType      string
	ISOClause        string
	Description      string
	Status           string
	CorrectiveAction *string
	DueDate          *time.Time
	ClosedAt         *time.Time
	CreatedAt        time.Time
}

// RiskEntry is one row from compliance.risk_register.
type RiskEntry struct {
	ID            uuid.UUID
	RiskTitle     string
	Likelihood    int
	Impact        int
	Treatment     string
	ResidualScore int
	OwnerID       *uuid.UUID
	ReviewDate    *time.Time
	CreatedAt     time.Time
}

// SupplierReview is one row from compliance.supplier_reviews.
type SupplierReview struct {
	ID              uuid.UUID
	VendorName      string
	ReviewStatus    string
	CertificateType *string
	CertificateURL  *string
	ReviewedAt      *time.Time
	NextReviewDue   *time.Time
	Notes           *string
	CreatedAt       time.Time
}

// TrainingCompletion is one row from compliance.security_training_completions.
type TrainingCompletion struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	TrainingYear int
	CompletedAt  time.Time
}

// SoAControlRow is one row from compliance.iso_soa_controls.
type SoAControlRow struct {
	ControlID              string
	Theme                  string
	Title                  string
	Status                 string
	ExclusionJustification *string
	UpdatedAt              time.Time
}

// ProgramStatus is the singleton compliance.isms_program_status row.
type ProgramStatus struct {
	ScopeStatement     string
	ISO27001Status     string
	ISO27001CertURL    *string
	ISO27001LastAudit  *time.Time
	ISO27701Status     string
	SoALastReview      *time.Time
	UpdatedAt          time.Time
}

// SoASummary holds aggregate SoA counts.
type SoASummary struct {
	Total      int
	Implemented int
	Planned    int
	Excluded   int
}

func ListAuditFindings(ctx context.Context, pool *pgxpool.Pool) ([]AuditFinding, error) {
	rows, err := pool.Query(ctx, `
SELECT id, audit_cycle, finding_type, iso_clause, description, status,
       corrective_action, due_date, closed_at, created_at
  FROM compliance.iso_audit_findings
 ORDER BY created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditFinding
	for rows.Next() {
		var f AuditFinding
		if err := rows.Scan(
			&f.ID, &f.AuditCycle, &f.FindingType, &f.ISOClause, &f.Description, &f.Status,
			&f.CorrectiveAction, &f.DueDate, &f.ClosedAt, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func InsertAuditFinding(ctx context.Context, pool *pgxpool.Pool, auditCycle, findingType, isoClause, description string, correctiveAction *string, dueDate *time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.iso_audit_findings (audit_cycle, finding_type, iso_clause, description, corrective_action, due_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`, auditCycle, findingType, isoClause, description, correctiveAction, dueDate).Scan(&id)
	return id, err
}

func UpdateAuditFinding(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, correctiveAction *string, dueDate *time.Time, closed bool) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE compliance.iso_audit_findings
   SET status = $2,
       corrective_action = COALESCE($3, corrective_action),
       due_date = COALESCE($4, due_date),
       closed_at = CASE WHEN $5 THEN COALESCE(closed_at, NOW()) ELSE closed_at END
 WHERE id = $1
`, id, status, correctiveAction, dueDate, closed)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func ListRiskEntries(ctx context.Context, pool *pgxpool.Pool) ([]RiskEntry, error) {
	rows, err := pool.Query(ctx, `
SELECT id, risk_title, likelihood, impact, treatment, residual_score, owner_id, review_date, created_at
  FROM compliance.risk_register
 ORDER BY residual_score DESC, created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RiskEntry
	for rows.Next() {
		var r RiskEntry
		if err := rows.Scan(
			&r.ID, &r.RiskTitle, &r.Likelihood, &r.Impact, &r.Treatment,
			&r.ResidualScore, &r.OwnerID, &r.ReviewDate, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func InsertRiskEntry(ctx context.Context, pool *pgxpool.Pool, title string, likelihood, impact int, treatment string, ownerID *uuid.UUID, reviewDate *time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.risk_register (risk_title, likelihood, impact, treatment, owner_id, review_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`, title, likelihood, impact, treatment, ownerID, reviewDate).Scan(&id)
	return id, err
}

func ListSupplierReviews(ctx context.Context, pool *pgxpool.Pool) ([]SupplierReview, error) {
	rows, err := pool.Query(ctx, `
SELECT id, vendor_name, review_status, certificate_type, certificate_url,
       reviewed_at, next_review_due, notes, created_at
  FROM compliance.supplier_reviews
 ORDER BY vendor_name
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SupplierReview
	for rows.Next() {
		var s SupplierReview
		if err := rows.Scan(
			&s.ID, &s.VendorName, &s.ReviewStatus, &s.CertificateType, &s.CertificateURL,
			&s.ReviewedAt, &s.NextReviewDue, &s.Notes, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func UpsertSupplierReview(ctx context.Context, pool *pgxpool.Pool, vendorName, reviewStatus string, certType, certURL, notes *string, reviewedAt *time.Time, nextReviewDue *time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.supplier_reviews (vendor_name, review_status, certificate_type, certificate_url, reviewed_at, next_review_due, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (vendor_name) DO UPDATE SET
  review_status = EXCLUDED.review_status,
  certificate_type = EXCLUDED.certificate_type,
  certificate_url = EXCLUDED.certificate_url,
  reviewed_at = EXCLUDED.reviewed_at,
  next_review_due = EXCLUDED.next_review_due,
  notes = EXCLUDED.notes
RETURNING id
`, vendorName, reviewStatus, certType, certURL, reviewedAt, nextReviewDue, notes).Scan(&id)
	return id, err
}

func RecordTrainingCompletion(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, year int) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.security_training_completions (user_id, training_year)
VALUES ($1, $2)
ON CONFLICT (user_id, training_year) DO UPDATE SET completed_at = NOW()
RETURNING id
`, userID, year).Scan(&id)
	return id, err
}

func ListTrainingCompletions(ctx context.Context, pool *pgxpool.Pool, year int) ([]TrainingCompletion, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, training_year, completed_at
  FROM compliance.security_training_completions
 WHERE training_year = $1
 ORDER BY completed_at DESC
`, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrainingCompletion
	for rows.Next() {
		var t TrainingCompletion
		if err := rows.Scan(&t.ID, &t.UserID, &t.TrainingYear, &t.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func SoAControlCount(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM compliance.iso_soa_controls`).Scan(&n)
	return n, err
}

func EnsureSoAControls(ctx context.Context, pool *pgxpool.Pool, controls []struct {
	ID, Theme, Title string
}) error {
	batch := &pgx.Batch{}
	for _, c := range controls {
		batch.Queue(`
INSERT INTO compliance.iso_soa_controls (control_id, theme, title, status)
VALUES ($1, $2, $3, 'planned')
ON CONFLICT (control_id) DO NOTHING
`, c.ID, c.Theme, c.Title)
	}
	br := pool.SendBatch(ctx, batch)
	for range controls {
		if _, err := br.Exec(); err != nil {
			closeErr := br.Close()
			if closeErr != nil {
				return errors.Join(err, closeErr)
			}
			return err
		}
	}
	return br.Close()
}

func ListSoAControls(ctx context.Context, pool *pgxpool.Pool) ([]SoAControlRow, error) {
	rows, err := pool.Query(ctx, `
SELECT control_id, theme, title, status, exclusion_justification, updated_at
  FROM compliance.iso_soa_controls
 ORDER BY control_id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SoAControlRow
	for rows.Next() {
		var c SoAControlRow
		if err := rows.Scan(&c.ControlID, &c.Theme, &c.Title, &c.Status, &c.ExclusionJustification, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func UpdateSoAControl(ctx context.Context, pool *pgxpool.Pool, controlID, status string, exclusion *string) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE compliance.iso_soa_controls
   SET status = $2,
       exclusion_justification = $3,
       updated_at = NOW()
 WHERE control_id = $1
`, controlID, status, exclusion)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func GetSoASummary(ctx context.Context, pool *pgxpool.Pool) (SoASummary, error) {
	var s SoASummary
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int,
       COUNT(*) FILTER (WHERE status = 'implemented')::int,
       COUNT(*) FILTER (WHERE status = 'planned')::int,
       COUNT(*) FILTER (WHERE status = 'excluded')::int
  FROM compliance.iso_soa_controls
`).Scan(&s.Total, &s.Implemented, &s.Planned, &s.Excluded)
	return s, err
}

func GetProgramStatus(ctx context.Context, pool *pgxpool.Pool) (ProgramStatus, error) {
	var p ProgramStatus
	err := pool.QueryRow(ctx, `
SELECT scope_statement, iso27001_status, iso27001_cert_url, iso27001_last_audit,
       iso27701_status, soa_last_review, updated_at
  FROM compliance.isms_program_status
 WHERE id = 1
`).Scan(
		&p.ScopeStatement, &p.ISO27001Status, &p.ISO27001CertURL, &p.ISO27001LastAudit,
		&p.ISO27701Status, &p.SoALastReview, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProgramStatus{}, ErrNotFound
	}
	return p, err
}

func UpdateProgramStatus(ctx context.Context, pool *pgxpool.Pool, scope, iso27001Status, iso27701Status string, certURL *string, lastAudit, soaReview *time.Time) error {
	_, err := pool.Exec(ctx, `
UPDATE compliance.isms_program_status
   SET scope_statement = $1,
       iso27001_status = $2,
       iso27001_cert_url = $3,
       iso27001_last_audit = $4,
       iso27701_status = $5,
       soa_last_review = $6,
       updated_at = NOW()
 WHERE id = 1
`, scope, iso27001Status, certURL, lastAudit, iso27701Status, soaReview)
	return err
}

var ErrNotFound = errors.New("iso: not found")
