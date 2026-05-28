// Package iso implements ISO/IEC 27001:2022 ISMS and ISO/IEC 27701:2019 PIMS tracking (plan 10.10).
package iso

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/iso"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// AdminPermission gates ISO ISMS admin APIs.
const AdminPermission = "compliance:iso:admin:*"

var (
	ErrNotFound      = errors.New("iso: record not found")
	ErrInvalidInput  = errors.New("iso: invalid input")
	ErrSoAIncomplete = errors.New("iso: statement of applicability not fully initialized")
)

// CheckAdmin returns true when the user holds compliance:iso:admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// EnsureSoAInitialized seeds all 93 Annex A controls when the table is empty.
func EnsureSoAInitialized(ctx context.Context, pool *pgxpool.Pool) error {
	n, err := repo.SoAControlCount(ctx, pool)
	if err != nil {
		return err
	}
	if n >= len(AnnexAControls) {
		return nil
	}
	seed := make([]struct{ ID, Theme, Title string }, len(AnnexAControls))
	for i, c := range AnnexAControls {
		seed[i] = struct{ ID, Theme, Title string }{c.ID, c.Theme, c.Title}
	}
	return repo.EnsureSoAControls(ctx, pool, seed)
}

// Dashboard aggregates ISMS program metrics for the compliance admin UI.
type Dashboard struct {
	Program         repo.ProgramStatus
	SoA             repo.SoASummary
	OpenFindings    int
	HighRisks       int
	PendingSuppliers int
	TrainingYear    int
	TrainingCount   int
}

func GetDashboard(ctx context.Context, pool *pgxpool.Pool) (Dashboard, error) {
	if err := EnsureSoAInitialized(ctx, pool); err != nil {
		return Dashboard{}, err
	}
	program, err := repo.GetProgramStatus(ctx, pool)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return Dashboard{}, err
	}
	soa, err := repo.GetSoASummary(ctx, pool)
	if err != nil {
		return Dashboard{}, err
	}
	findings, err := repo.ListAuditFindings(ctx, pool)
	if err != nil {
		return Dashboard{}, err
	}
	openFindings := 0
	for _, f := range findings {
		if f.Status != "closed" {
			openFindings++
		}
	}
	risks, err := repo.ListRiskEntries(ctx, pool)
	if err != nil {
		return Dashboard{}, err
	}
	highRisks := 0
	for _, r := range risks {
		if r.ResidualScore >= 15 {
			highRisks++
		}
	}
	suppliers, err := repo.ListSupplierReviews(ctx, pool)
	if err != nil {
		return Dashboard{}, err
	}
	pendingSuppliers := 0
	for _, s := range suppliers {
		if s.ReviewStatus == "pending" {
			pendingSuppliers++
		}
	}
	year := time.Now().UTC().Year()
	training, err := repo.ListTrainingCompletions(ctx, pool, year)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		Program:          program,
		SoA:              soa,
		OpenFindings:     openFindings,
		HighRisks:        highRisks,
		PendingSuppliers: pendingSuppliers,
		TrainingYear:     year,
		TrainingCount:    len(training),
	}, nil
}

func ListAuditFindings(ctx context.Context, pool *pgxpool.Pool) ([]repo.AuditFinding, error) {
	return repo.ListAuditFindings(ctx, pool)
}

func CreateAuditFinding(ctx context.Context, pool *pgxpool.Pool, auditCycle, findingType, isoClause, description string, correctiveAction *string, dueDate *time.Time) (uuid.UUID, error) {
	if !validFindingType(findingType) {
		return uuid.UUID{}, ErrInvalidInput
	}
	return repo.InsertAuditFinding(ctx, pool, auditCycle, findingType, isoClause, description, correctiveAction, dueDate)
}

func PatchAuditFinding(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, correctiveAction *string, dueDate *time.Time, closeFinding bool) error {
	if status != "" && status != "open" && status != "in_progress" && status != "closed" {
		return ErrInvalidInput
	}
	if status == "" {
		status = "open"
	}
	ok, err := repo.UpdateAuditFinding(ctx, pool, id, status, correctiveAction, dueDate, closeFinding || status == "closed")
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func ListRiskEntries(ctx context.Context, pool *pgxpool.Pool) ([]repo.RiskEntry, error) {
	return repo.ListRiskEntries(ctx, pool)
}

func CreateRiskEntry(ctx context.Context, pool *pgxpool.Pool, title string, likelihood, impact int, treatment string, ownerID *uuid.UUID, reviewDate *time.Time) (uuid.UUID, error) {
	if likelihood < 1 || likelihood > 5 || impact < 1 || impact > 5 {
		return uuid.UUID{}, ErrInvalidInput
	}
	if !validTreatment(treatment) {
		return uuid.UUID{}, ErrInvalidInput
	}
	return repo.InsertRiskEntry(ctx, pool, title, likelihood, impact, treatment, ownerID, reviewDate)
}

func ListSupplierReviews(ctx context.Context, pool *pgxpool.Pool) ([]repo.SupplierReview, error) {
	return repo.ListSupplierReviews(ctx, pool)
}

func UpsertSupplierReview(ctx context.Context, pool *pgxpool.Pool, vendorName, reviewStatus string, certType, certURL, notes *string, reviewedAt, nextReviewDue *time.Time) (uuid.UUID, error) {
	if vendorName == "" || !validReviewStatus(reviewStatus) {
		return uuid.UUID{}, ErrInvalidInput
	}
	return repo.UpsertSupplierReview(ctx, pool, vendorName, reviewStatus, certType, certURL, notes, reviewedAt, nextReviewDue)
}

func RecordTraining(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, year int) (uuid.UUID, error) {
	if year < 2000 || year > 2100 {
		return uuid.UUID{}, ErrInvalidInput
	}
	return repo.RecordTrainingCompletion(ctx, pool, userID, year)
}

func ListTraining(ctx context.Context, pool *pgxpool.Pool, year int) ([]repo.TrainingCompletion, error) {
	return repo.ListTrainingCompletions(ctx, pool, year)
}

func ListSoAControls(ctx context.Context, pool *pgxpool.Pool) ([]repo.SoAControlRow, error) {
	if err := EnsureSoAInitialized(ctx, pool); err != nil {
		return nil, err
	}
	return repo.ListSoAControls(ctx, pool)
}

func PatchSoAControl(ctx context.Context, pool *pgxpool.Pool, controlID, status string, exclusion *string) error {
	if status != "implemented" && status != "planned" && status != "excluded" {
		return ErrInvalidInput
	}
	if status == "excluded" && (exclusion == nil || *exclusion == "") {
		return fmt.Errorf("%w: exclusion justification required", ErrInvalidInput)
	}
	ok, err := repo.UpdateSoAControl(ctx, pool, controlID, status, exclusion)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func GetTrustProgram(ctx context.Context, pool *pgxpool.Pool) (repo.ProgramStatus, repo.SoASummary, error) {
	if err := EnsureSoAInitialized(ctx, pool); err != nil {
		return repo.ProgramStatus{}, repo.SoASummary{}, err
	}
	program, err := repo.GetProgramStatus(ctx, pool)
	if err != nil {
		return repo.ProgramStatus{}, repo.SoASummary{}, err
	}
	soa, err := repo.GetSoASummary(ctx, pool)
	if err != nil {
		return repo.ProgramStatus{}, repo.SoASummary{}, err
	}
	return program, soa, nil
}

func UpdateProgramStatus(ctx context.Context, pool *pgxpool.Pool, scope, iso27001Status, iso27701Status string, certURL *string, lastAudit, soaReview *time.Time) error {
	if scope == "" {
		return ErrInvalidInput
	}
	return repo.UpdateProgramStatus(ctx, pool, scope, iso27001Status, iso27701Status, certURL, lastAudit, soaReview)
}

func validFindingType(t string) bool {
	switch t {
	case "nonconformity", "observation", "opportunity":
		return true
	default:
		return false
	}
}

func validTreatment(t string) bool {
	switch t {
	case "mitigate", "accept", "transfer", "avoid":
		return true
	default:
		return false
	}
}

func validReviewStatus(s string) bool {
	switch s {
	case "pending", "approved", "rejected":
		return true
	default:
		return false
	}
}
