// Package securityreports persists responsible-disclosure vulnerability reports (plan 10.16).
package securityreports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Report is one row from compliance.security_reports.
type Report struct {
	ID             uuid.UUID
	ReporterHandle *string
	ReportDate     time.Time
	TriagedAt      *time.Time
	CVSSScore      *float64
	Severity       *string
	Summary        string
	Status         string
	PatchDate      *time.Time
	SLAMet         *bool
	BountyPaid     bool
	CreatedAt      time.Time
}

// InsertReport creates a new vulnerability report.
func InsertReport(ctx context.Context, pool *pgxpool.Pool, reporterHandle *string, reportDate time.Time, cvssScore *float64, severity *string, summary string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.security_reports (reporter_handle, report_date, cvss_score, severity, summary)
VALUES ($1, $2::date, $3, $4, $5)
RETURNING id
`, reporterHandle, reportDate, cvssScore, severity, summary).Scan(&id)
	return id, err
}

// ListReports returns reports ordered by report_date DESC, capped at limit.
func ListReports(ctx context.Context, pool *pgxpool.Pool, limit int) ([]Report, error) {
	rows, err := pool.Query(ctx, `
SELECT id, reporter_handle, report_date, triaged_at, cvss_score, severity, summary, status, patch_date, sla_met, bounty_paid, created_at
  FROM compliance.security_reports
 ORDER BY report_date DESC, created_at DESC
 LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReports(rows)
}

// GetReport returns a report by ID or nil if not found.
func GetReport(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Report, error) {
	row := pool.QueryRow(ctx, `
SELECT id, reporter_handle, report_date, triaged_at, cvss_score, severity, summary, status, patch_date, sla_met, bounty_paid, created_at
  FROM compliance.security_reports
 WHERE id = $1
`, id)
	r, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// UpdateReport patches triage fields and optional resolution details.
func UpdateReport(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, severity *string, cvssScore *float64, triagedAt *time.Time, patchDate *time.Time, slaMet *bool, bountyPaid *bool) error {
	_, err := pool.Exec(ctx, `
UPDATE compliance.security_reports
   SET status          = $2,
       severity        = COALESCE($3, severity),
       cvss_score      = COALESCE($4, cvss_score),
       triaged_at      = COALESCE($5, triaged_at),
       patch_date      = COALESCE($6, patch_date),
       sla_met         = COALESCE($7, sla_met),
       bounty_paid     = COALESCE($8, bounty_paid)
 WHERE id = $1
`, id, status, severity, cvssScore, triagedAt, patchDate, slaMet, bountyPaid)
	return err
}

func scanReports(rows pgx.Rows) ([]Report, error) {
	var out []Report
	for rows.Next() {
		r, err := scanReportRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanReport(row pgx.Row) (*Report, error) {
	r, err := scanReportRow(row)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func scanReportRow(row pgx.Row) (Report, error) {
	var r Report
	err := row.Scan(
		&r.ID, &r.ReporterHandle, &r.ReportDate, &r.TriagedAt, &r.CVSSScore, &r.Severity,
		&r.Summary, &r.Status, &r.PatchDate, &r.SLAMet, &r.BountyPaid, &r.CreatedAt,
	)
	return r, err
}
