// Package soc2 persists SOC 2 Type II compliance data: access reviews,
// incident records, and the vendor risk register (plan 10.9).
package soc2

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccessReview is one row from compliance.access_reviews.
type AccessReview struct {
	ID             uuid.UUID
	ReviewerID     uuid.UUID
	ReviewType     string
	ReviewedAt     time.Time
	Findings       *string // JSON blob
	NextReviewDue  *time.Time
}

// Incident is one row from compliance.incidents.
type Incident struct {
	ID             uuid.UUID
	Title          string
	Severity       string
	Status         string
	OpenedAt       time.Time
	ResolvedAt     *time.Time
	PostMortemURL  *string
	TSCCriteria    []string
}

// VendorRisk is one row from compliance.vendor_risk.
type VendorRisk struct {
	ID             uuid.UUID
	VendorName     string
	SOC2ReportURL  *string
	ReportDate     *time.Time
	RiskTier       string
	NextReviewDue  *time.Time
	Notes          *string
}

// InsertAccessReview creates a new access review record.
func InsertAccessReview(ctx context.Context, pool *pgxpool.Pool, reviewerID uuid.UUID, reviewType string, findings *string, nextReviewDue *time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.access_reviews (reviewer_id, review_type, findings, next_review_due)
VALUES ($1, $2, $3::jsonb, $4)
RETURNING id
`, reviewerID, reviewType, findings, nextReviewDue).Scan(&id)
	return id, err
}

// ListAccessReviews returns access reviews ordered by most recent, capped at limit.
func ListAccessReviews(ctx context.Context, pool *pgxpool.Pool, limit int) ([]AccessReview, error) {
	rows, err := pool.Query(ctx, `
SELECT id, reviewer_id, review_type, reviewed_at, findings::text, next_review_due
  FROM compliance.access_reviews
 ORDER BY reviewed_at DESC
 LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccessReview
	for rows.Next() {
		var r AccessReview
		if err := rows.Scan(&r.ID, &r.ReviewerID, &r.ReviewType, &r.ReviewedAt, &r.Findings, &r.NextReviewDue); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// InsertIncident creates a new incident record and returns its ID.
func InsertIncident(ctx context.Context, pool *pgxpool.Pool, title, severity string, tscCriteria []string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.incidents (title, severity, tsc_criteria)
VALUES ($1, $2, $3)
RETURNING id
`, title, severity, tscCriteria).Scan(&id)
	return id, err
}

// GetIncident returns a single incident by ID, or nil if not found.
func GetIncident(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Incident, error) {
	var inc Incident
	err := pool.QueryRow(ctx, `
SELECT id, title, severity, status, opened_at, resolved_at, post_mortem_url, tsc_criteria
  FROM compliance.incidents
 WHERE id = $1
`, id).Scan(&inc.ID, &inc.Title, &inc.Severity, &inc.Status, &inc.OpenedAt, &inc.ResolvedAt, &inc.PostMortemURL, &inc.TSCCriteria)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inc, nil
}

// ListIncidents returns incidents filtered by status (empty = all), ordered by opened_at DESC.
func ListIncidents(ctx context.Context, pool *pgxpool.Pool, status string, limit int) ([]Incident, error) {
	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = pool.Query(ctx, `
SELECT id, title, severity, status, opened_at, resolved_at, post_mortem_url, tsc_criteria
  FROM compliance.incidents
 WHERE status = $1
 ORDER BY opened_at DESC
 LIMIT $2
`, status, limit)
	} else {
		rows, err = pool.Query(ctx, `
SELECT id, title, severity, status, opened_at, resolved_at, post_mortem_url, tsc_criteria
  FROM compliance.incidents
 ORDER BY opened_at DESC
 LIMIT $1
`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Incident
	for rows.Next() {
		var inc Incident
		if err := rows.Scan(&inc.ID, &inc.Title, &inc.Severity, &inc.Status, &inc.OpenedAt, &inc.ResolvedAt, &inc.PostMortemURL, &inc.TSCCriteria); err != nil {
			return nil, err
		}
		out = append(out, inc)
	}
	return out, rows.Err()
}

// UpdateIncident transitions an incident to a new status and sets optional resolved_at / post_mortem_url.
func UpdateIncident(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, resolvedAt *time.Time, postMortemURL *string) error {
	_, err := pool.Exec(ctx, `
UPDATE compliance.incidents
   SET status          = $2,
       resolved_at     = COALESCE($3, resolved_at),
       post_mortem_url = COALESCE($4, post_mortem_url)
 WHERE id = $1
`, id, status, resolvedAt, postMortemURL)
	return err
}

// UpsertVendor inserts or updates a vendor risk row by vendor_name.
func UpsertVendor(ctx context.Context, pool *pgxpool.Pool, name, riskTier string, soc2ReportURL *string, reportDate *time.Time, nextReviewDue *time.Time, notes *string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.vendor_risk (vendor_name, soc2_report_url, report_date, risk_tier, next_review_due, notes)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (vendor_name) DO UPDATE
   SET soc2_report_url = EXCLUDED.soc2_report_url,
       report_date     = EXCLUDED.report_date,
       risk_tier       = EXCLUDED.risk_tier,
       next_review_due = EXCLUDED.next_review_due,
       notes           = EXCLUDED.notes
RETURNING id
`, name, soc2ReportURL, reportDate, riskTier, nextReviewDue, notes).Scan(&id)
	return id, err
}

// ListVendors returns all vendor risk rows ordered by risk_tier severity then vendor_name.
func ListVendors(ctx context.Context, pool *pgxpool.Pool) ([]VendorRisk, error) {
	rows, err := pool.Query(ctx, `
SELECT id, vendor_name, soc2_report_url, report_date, risk_tier, next_review_due, notes
  FROM compliance.vendor_risk
 ORDER BY
   CASE risk_tier WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
   vendor_name
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VendorRisk
	for rows.Next() {
		var v VendorRisk
		if err := rows.Scan(&v.ID, &v.VendorName, &v.SOC2ReportURL, &v.ReportDate, &v.RiskTier, &v.NextReviewDue, &v.Notes); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
