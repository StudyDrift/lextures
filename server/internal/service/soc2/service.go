// Package soc2 implements SOC 2 Type II compliance: access reviews,
// incident response, and vendor risk management (plan 10.9;
// AICPA Trust Services Criteria 2017, SSAE 18 / AT-C section 320).
package soc2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/soc2"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// AdminPermission gates all SOC 2 compliance admin actions (CC6.3).
const AdminPermission = "compliance:soc2:admin:*"

var (
	ErrNotFound  = errors.New("soc2: record not found")
	ErrForbidden = errors.New("soc2: forbidden")
)

// CheckAdmin returns true when the user holds the soc2 admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// CreateAccessReview records a completed access review (FR-2, AC-2).
func CreateAccessReview(ctx context.Context, pool *pgxpool.Pool, reviewerID uuid.UUID, reviewType string, findings *string, nextReviewDue *time.Time) (uuid.UUID, error) {
	id, err := repo.InsertAccessReview(ctx, pool, reviewerID, reviewType, findings, nextReviewDue)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("soc2: create access review: %w", err)
	}
	return id, nil
}

// ListAccessReviews returns recent access reviews for the evidence dashboard (AC-2).
func ListAccessReviews(ctx context.Context, pool *pgxpool.Pool) ([]repo.AccessReview, error) {
	reviews, err := repo.ListAccessReviews(ctx, pool, 200)
	if err != nil {
		return nil, fmt.Errorf("soc2: list access reviews: %w", err)
	}
	return reviews, nil
}

// OpenIncident creates a new security incident record (FR-4, AC-3).
func OpenIncident(ctx context.Context, pool *pgxpool.Pool, title, severity string, tscCriteria []string) (uuid.UUID, error) {
	id, err := repo.InsertIncident(ctx, pool, title, severity, tscCriteria)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("soc2: open incident: %w", err)
	}
	return id, nil
}

// GetIncident returns an incident by ID or ErrNotFound.
func GetIncident(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*repo.Incident, error) {
	inc, err := repo.GetIncident(ctx, pool, id)
	if err != nil {
		return nil, fmt.Errorf("soc2: get incident: %w", err)
	}
	if inc == nil {
		return nil, ErrNotFound
	}
	return inc, nil
}

// ListIncidents returns incidents filtered by status ("" = all) (AC-3).
func ListIncidents(ctx context.Context, pool *pgxpool.Pool, status string) ([]repo.Incident, error) {
	incidents, err := repo.ListIncidents(ctx, pool, status, 500)
	if err != nil {
		return nil, fmt.Errorf("soc2: list incidents: %w", err)
	}
	return incidents, nil
}

// UpdateIncidentStatus transitions an incident and optionally records resolution details.
func UpdateIncidentStatus(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, resolvedAt *time.Time, postMortemURL *string) error {
	// Auto-set resolved_at for terminal statuses when not provided.
	if (status == "resolved" || status == "closed") && resolvedAt == nil {
		now := time.Now().UTC()
		resolvedAt = &now
	}
	if err := repo.UpdateIncident(ctx, pool, id, status, resolvedAt, postMortemURL); err != nil {
		return fmt.Errorf("soc2: update incident: %w", err)
	}
	return nil
}

// UpsertVendor adds or refreshes a vendor in the risk register (FR-6, AC-6).
func UpsertVendor(ctx context.Context, pool *pgxpool.Pool, name, riskTier string, soc2ReportURL *string, reportDate *time.Time, nextReviewDue *time.Time, notes *string) (uuid.UUID, error) {
	id, err := repo.UpsertVendor(ctx, pool, name, riskTier, soc2ReportURL, reportDate, nextReviewDue, notes)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("soc2: upsert vendor: %w", err)
	}
	return id, nil
}

// ListVendors returns the full vendor risk register.
func ListVendors(ctx context.Context, pool *pgxpool.Pool) ([]repo.VendorRisk, error) {
	vendors, err := repo.ListVendors(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("soc2: list vendors: %w", err)
	}
	return vendors, nil
}

// EvidenceSummary is the overview shown on the compliance dashboard.
type EvidenceSummary struct {
	OpenIncidents      int  `json:"openIncidents"`
	RecentReviews      int  `json:"recentReviews"`
	VendorsTotal       int  `json:"vendorsTotal"`
	VendorsOverdue     int  `json:"vendorsOverdue"`
}

// GetEvidenceSummary returns counts for the compliance dashboard.
func GetEvidenceSummary(ctx context.Context, pool *pgxpool.Pool) (EvidenceSummary, error) {
	incidents, err := repo.ListIncidents(ctx, pool, "open", 1000)
	if err != nil {
		return EvidenceSummary{}, fmt.Errorf("soc2: evidence summary: %w", err)
	}

	// Count contained as open too (still active).
	openCount := 0
	for _, inc := range incidents {
		if inc.Status == "open" || inc.Status == "contained" {
			openCount++
		}
	}

	reviews, err := repo.ListAccessReviews(ctx, pool, 1000)
	if err != nil {
		return EvidenceSummary{}, fmt.Errorf("soc2: evidence summary reviews: %w", err)
	}

	vendors, err := repo.ListVendors(ctx, pool)
	if err != nil {
		return EvidenceSummary{}, fmt.Errorf("soc2: evidence summary vendors: %w", err)
	}

	overdue := 0
	now := time.Now().UTC()
	for _, v := range vendors {
		if v.NextReviewDue != nil && v.NextReviewDue.Before(now) {
			overdue++
		}
	}

	return EvidenceSummary{
		OpenIncidents:  openCount,
		RecentReviews:  len(reviews),
		VendorsTotal:   len(vendors),
		VendorsOverdue: overdue,
	}, nil
}
