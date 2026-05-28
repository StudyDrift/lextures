// Package securityreports implements responsible-disclosure intake and triage (plan 10.16).
package securityreports

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/securityreports"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// AdminPermission gates security report admin APIs.
const AdminPermission = "compliance:security:admin:*"

var ErrNotFound = errors.New("securityreports: not found")

// CheckAdmin returns true when the user holds the security disclosure admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// PatchSLADays returns the patch SLA in calendar days for a CVSS severity tier (FR-4).
func PatchSLADays(severity string) int {
	switch severity {
	case "critical":
		return 7
	case "high":
		return 30
	case "medium":
		return 90
	default:
		return 0
	}
}

// ComputeSLAMet returns whether patch_date met the SLA for the given severity, or nil when not applicable.
func ComputeSLAMet(reportDate time.Time, patchDate *time.Time, severity string) *bool {
	if patchDate == nil || severity == "" {
		return nil
	}
	days := PatchSLADays(severity)
	if days == 0 {
		return nil
	}
	deadline := reportDate.UTC().Truncate(24 * time.Hour).AddDate(0, 0, days)
	patch := patchDate.UTC().Truncate(24 * time.Hour)
	met := !patch.After(deadline)
	return &met
}

// CreateReport logs an incoming vulnerability report.
func CreateReport(ctx context.Context, pool *pgxpool.Pool, reporterHandle *string, reportDate time.Time, cvssScore *float64, severity *string, summary string) (uuid.UUID, error) {
	if summary == "" {
		return uuid.UUID{}, fmt.Errorf("securityreports: summary required")
	}
	id, err := repo.InsertReport(ctx, pool, reporterHandle, reportDate, cvssScore, severity, summary)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("securityreports: create: %w", err)
	}
	return id, nil
}

// ListReports returns recent reports for the compliance admin UI.
func ListReports(ctx context.Context, pool *pgxpool.Pool) ([]repo.Report, error) {
	reports, err := repo.ListReports(ctx, pool, 500)
	if err != nil {
		return nil, fmt.Errorf("securityreports: list: %w", err)
	}
	return reports, nil
}

// GetReport returns a report by ID.
func GetReport(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*repo.Report, error) {
	r, err := repo.GetReport(ctx, pool, id)
	if err != nil {
		return nil, fmt.Errorf("securityreports: get: %w", err)
	}
	if r == nil {
		return nil, ErrNotFound
	}
	return r, nil
}

// UpdateReportInput is the service-layer patch payload.
type UpdateReportInput struct {
	Status     string
	Severity   *string
	CVSSScore  *float64
	TriagedAt  *time.Time
	PatchDate  *time.Time
	BountyPaid *bool
}

// UpdateReport updates triage status and recalculates SLA adherence when patched.
func UpdateReport(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, in UpdateReportInput) error {
	existing, err := repo.GetReport(ctx, pool, id)
	if err != nil {
		return fmt.Errorf("securityreports: update load: %w", err)
	}
	if existing == nil {
		return ErrNotFound
	}

	severity := existing.Severity
	if in.Severity != nil {
		severity = in.Severity
	}

	triagedAt := existing.TriagedAt
	if in.TriagedAt != nil {
		triagedAt = in.TriagedAt
	} else if in.Status == "accepted" && triagedAt == nil {
		now := time.Now().UTC()
		triagedAt = &now
	}

	patchDate := existing.PatchDate
	if in.PatchDate != nil {
		patchDate = in.PatchDate
	} else if in.Status == "patched" && patchDate == nil {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		patchDate = &today
	}

	var slaMet *bool
	if in.Status == "patched" && severity != nil {
		slaMet = ComputeSLAMet(existing.ReportDate, patchDate, *severity)
	}

	bountyPaid := existing.BountyPaid
	if in.BountyPaid != nil {
		bountyPaid = *in.BountyPaid
	}

	if err := repo.UpdateReport(ctx, pool, id, in.Status, in.Severity, in.CVSSScore, triagedAt, patchDate, slaMet, &bountyPaid); err != nil {
		return fmt.Errorf("securityreports: update: %w", err)
	}
	return nil
}

// TrustPolicy is the public responsible-disclosure summary (FR-1, AC-5).
type TrustPolicy struct {
	ContactEmail           string  `json:"contactEmail"`
	PGPFingerprint         string  `json:"pgpFingerprint"`
	PGPKeyURL              string  `json:"pgpKeyUrl"`
	CoordinatedDisclosureDays int  `json:"coordinatedDisclosureDays"`
	PatchSLADays           map[string]int `json:"patchSlaDays"`
	PolicyPagePath         string  `json:"policyPagePath"`
	RepositorySecurityPath string  `json:"repositorySecurityPath"`
}

// DefaultTrustPolicy returns the published disclosure policy metadata.
func DefaultTrustPolicy() TrustPolicy {
	return TrustPolicy{
		ContactEmail:              "security@lextures.io",
		PGPFingerprint:            "E3F4 9A12 7B6C 8D01 4F2E 91A3 5C7D 0E8B 2A4F 6B9C",
		PGPKeyURL:                 "https://keys.openpgp.org/search?q=security%40lextures.io",
		CoordinatedDisclosureDays: 90,
		PolicyPagePath:            "/security",
		RepositorySecurityPath:    "SECURITY.md",
		PatchSLADays: map[string]int{
			"critical": 7,
			"high":     30,
			"medium":   90,
		},
	}
}
