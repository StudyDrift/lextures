// Package gdpr implements GDPR/UK GDPR compliance: DSAR workflow, consent management,
// RoPA, and right-to-erasure (plan 10.3; GDPR Art. 6, 7, 15, 17, 20, 30).
package gdpr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"

	pkgai "github.com/lextures/lextures/server/internal/aidisclosure"
	repoaidisclosure "github.com/lextures/lextures/server/internal/repos/aidisclosure"
	repo "github.com/lextures/lextures/server/internal/repos/gdpr"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// AdminPermission gates GDPR admin actions (approve/reject DSARs, manage RoPA).
const AdminPermission = "compliance:gdpr:admin:*"

// DPOPermission gates Data Protection Officer actions (RoPA export, DPA template).
const DPOPermission = "compliance:gdpr:dpo:*"

// DSARDeadlineWarning is the window before due_at at which an escalation is sent.
const DSARDeadlineWarning = 5 * 24 * time.Hour

// ArchiveLinkTTL is how long a completed DSAR download link is valid.
const ArchiveLinkTTL = 72 * time.Hour

// Sentinel purposes defined for AI consent gating (AC-3).
const PurposeAIProcessing = "ai_processing"

var (
	ErrNotFound      = errors.New("gdpr: record not found")
	ErrAlreadyExists = errors.New("gdpr: active request already exists")
	ErrForbidden     = errors.New("gdpr: forbidden")
)

// CheckAdmin returns true when the user holds the compliance:gdpr:admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// CheckDPO returns true when the user holds the compliance:gdpr:dpo permission.
func CheckDPO(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, DPOPermission)
}

// GrantConsent records a new consent grant for the given user and purpose.
// ipHash should be a SHA-256 of the request IP (or nil when unavailable).
func GrantConsent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, purpose, lawfulBasis, version string, ipHash *string) (uuid.UUID, error) {
	return repo.InsertConsent(ctx, pool, userID, purpose, lawfulBasis, version, ipHash)
}

// WithdrawConsent marks an existing consent record as withdrawn (AC-3).
// Returns ErrNotFound when no active consent matching id+userID exists.
func WithdrawConsent(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID) error {
	ok, err := repo.WithdrawConsent(ctx, pool, id, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// ListConsents returns all consent records for the user.
func ListConsents(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]repo.ConsentRecord, error) {
	return repo.ListConsents(ctx, pool, userID)
}

// IsAIConsentActive returns true when the user has an active (non-withdrawn) ai_processing consent.
// When false, AI feature routes must return HTTP 403 before assembling any prompt (AC-3).
func IsAIConsentActive(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return repo.HasActiveConsent(ctx, pool, userID, PurposeAIProcessing)
}

// SubmitDSAR creates a new DSAR for the authenticated user.
// Returns ErrAlreadyExists when the user already has a pending/in-progress DSAR of the same type.
func SubmitDSAR(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, userID uuid.UUID, requestType string) (uuid.UUID, error) {
	existing, err := repo.ListDSARRequestsForUser(ctx, pool, userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	for _, r := range existing {
		if r.RequestType == requestType && (r.Status == "pending" || r.Status == "in_progress") {
			return uuid.UUID{}, ErrAlreadyExists
		}
	}
	return repo.InsertDSARRequest(ctx, pool, orgID, userID, requestType)
}

// GetDSARForUser returns a DSAR request that belongs to the given user, or ErrNotFound.
func GetDSARForUser(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID) (*repo.DSARRequest, error) {
	r, err := repo.GetDSARRequest(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if r == nil || r.UserID != userID {
		return nil, ErrNotFound
	}
	return r, nil
}

// ListDSARsForUser returns all DSAR requests submitted by the user.
func ListDSARsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]repo.DSARRequest, error) {
	return repo.ListDSARRequestsForUser(ctx, pool, userID)
}

// ListPendingDSARs returns all pending/in-progress requests for admin review.
func ListPendingDSARs(ctx context.Context, pool *pgxpool.Pool) ([]repo.DSARRequest, error) {
	return repo.ListDSARRequestsPending(ctx, pool, 500)
}

// ApproveDSAR transitions a DSAR to in_progress and compiles an archive for access/portability requests.
// For erasure requests, it triggers anonymisation of user data.
func ApproveDSAR(ctx context.Context, pool *pgxpool.Pool, id, adminID uuid.UUID) error {
	r, err := repo.GetDSARRequest(ctx, pool, id)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrNotFound
	}

	switch r.RequestType {
	case "erasure":
		// Anonymise user data and mark completed.
		if err := repo.AnonymiseUser(ctx, pool, r.UserID); err != nil {
			return fmt.Errorf("gdpr: anonymise user: %w", err)
		}
		t := time.Now().UTC()
		expiresAt := t.Add(ArchiveLinkTTL)
		archiveURL := buildErasureConfirmationURL(id)
		return repo.UpdateDSARStatus(ctx, pool, id, adminID, "completed", &archiveURL, &expiresAt, nil)
	default:
		// For access / portability: compile archive synchronously.
		archiveJSON, err := compileUserArchive(ctx, pool, r.UserID)
		if err != nil {
			return fmt.Errorf("gdpr: compile archive: %w", err)
		}
		expiry := time.Now().UTC().Add(ArchiveLinkTTL)
		return repo.UpdateDSARStatus(ctx, pool, id, adminID, "completed", &archiveJSON, &expiry, nil)
	}
}

// RejectDSAR marks a request as rejected with a reason.
func RejectDSAR(ctx context.Context, pool *pgxpool.Pool, id, adminID uuid.UUID, reason string) error {
	r, err := repo.GetDSARRequest(ctx, pool, id)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrNotFound
	}
	return repo.UpdateDSARStatus(ctx, pool, id, adminID, "rejected", nil, nil, &reason)
}

// CountOverdueDSARs returns how many DSARs are past their 30-day deadline.
func CountOverdueDSARs(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	return repo.CountOverdueDSARs(ctx, pool)
}

// ListDSARsDueSoon returns requests expiring within DSARDeadlineWarning (5 days).
func ListDSARsDueSoon(ctx context.Context, pool *pgxpool.Pool) ([]repo.DSARRequest, error) {
	return repo.ListDSARsDueSoon(ctx, pool, DSARDeadlineWarning)
}

// AddRoPAEntry inserts a Record of Processing Activity.
func AddRoPAEntry(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, activityName, purpose, lawfulBasis, retentionPeriod string, dataCategories, dataSubjects, subProcessors []string) (uuid.UUID, error) {
	return repo.InsertRoPAEntry(ctx, pool, orgID, activityName, purpose, lawfulBasis, retentionPeriod, dataCategories, dataSubjects, subProcessors)
}

// ListRoPAEntries returns all RoPA entries for an org.
func ListRoPAEntries(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]repo.RoPAEntry, error) {
	return repo.ListRoPAEntries(ctx, pool, orgID)
}

// DeleteRoPAEntry removes a RoPA entry for an org.
func DeleteRoPAEntry(ctx context.Context, pool *pgxpool.Pool, id, orgID uuid.UUID) error {
	ok, err := repo.DeleteRoPAEntry(ctx, pool, id, orgID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// DPATemplateData is the pre-populated fields for a Data Processing Agreement (Art. 28).
type DPATemplateData struct {
	ControllerName     string   `json:"controllerName"`
	ProcessorName      string   `json:"processorName"`
	PrivacyPolicyURL   string   `json:"privacyPolicyUrl"`
	SubProcessors      []string `json:"subProcessors"`
	ProcessingPurposes []string `json:"processingPurposes"`
	TechnicalSafeguards []string `json:"technicalSafeguards"`
	GeneratedAt        string   `json:"generatedAt"`
}

// GenerateDPATemplate returns a pre-populated DPA template for an institutional admin.
func GenerateDPATemplate(orgName, privacyPolicyURL string) *DPATemplateData {
	return &DPATemplateData{
		ControllerName:   orgName,
		ProcessorName:    "Lextures, Inc.",
		PrivacyPolicyURL: privacyPolicyURL,
		SubProcessors: []string{
			"OpenRouter (AI model routing)",
			"AWS (cloud infrastructure)",
			"SendGrid (transactional email)",
		},
		ProcessingPurposes: []string{
			"Educational course delivery and assessment",
			"AI-assisted tutoring and feedback (with consent)",
			"Analytics and adaptive learning",
			"Legal compliance and fraud prevention",
		},
		TechnicalSafeguards: []string{
			"AES-256 encryption at rest",
			"TLS 1.3 in transit",
			"Role-based access control",
			"Audit logging for all data access",
			"Annual penetration testing",
		},
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// compileUserArchive fetches user data and returns a JSON string for the DSAR archive.
// In production this would stream to object storage; here it returns inline JSON.
func compileUserArchive(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	type profileRow struct {
		Email       string  `json:"email"`
		DisplayName string  `json:"displayName"`
		FirstName   *string `json:"firstName,omitempty"`
		LastName    *string `json:"lastName,omitempty"`
		CreatedAt   string  `json:"createdAt"`
	}
	var p profileRow
	err := pool.QueryRow(ctx, `
SELECT email, display_name, first_name, last_name, created_at
  FROM "user".users WHERE id = $1
`, userID).Scan(&p.Email, &p.DisplayName, &p.FirstName, &p.LastName, &p.CreatedAt)
	if err != nil {
		return "", err
	}

	consents, err := repo.ListConsents(ctx, pool, userID)
	if err != nil {
		return "", err
	}

	type archiveDoc struct {
		UserID            string           `json:"userId"`
		Profile           profileRow       `json:"profile"`
		Consents          []consentSummary `json:"consents"`
		AIInferenceLog    []map[string]any `json:"aiInferenceLog,omitempty"`
		ExportedAt        string           `json:"exportedAt"`
	}

	cs := make([]consentSummary, 0, len(consents))
	for _, c := range consents {
		cs = append(cs, consentSummary{
			ID:      c.ID.String(),
			Purpose: c.Purpose,
			Basis:   c.LawfulBasis,
			Version: c.ConsentVersion,
			Granted: c.GrantedAt.UTC().Format(time.RFC3339),
			Withdrawn: optRFC3339(c.WithdrawnAt),
		})
	}

	aiLog := dsarAIInferenceSummary(ctx, pool, os.Getenv("JWT_SECRET"), userID)

	doc := archiveDoc{
		UserID:         userID.String(),
		Profile:        p,
		Consents:       cs,
		AIInferenceLog: aiLog,
		ExportedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type consentSummary struct {
	ID        string  `json:"id"`
	Purpose   string  `json:"purpose"`
	Basis     string  `json:"lawfulBasis"`
	Version   string  `json:"consentVersion"`
	Granted   string  `json:"grantedAt"`
	Withdrawn *string `json:"withdrawnAt,omitempty"`
}

func optRFC3339(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

// buildErasureConfirmationURL returns a token representing the completed erasure.
// In a full implementation this would point to a confirmation page.
func buildErasureConfirmationURL(requestID uuid.UUID) string {
	return "erasure-confirmed:" + requestID.String()
}

func dsarAIInferenceSummary(ctx context.Context, pool *pgxpool.Pool, secret string, userID uuid.UUID) []map[string]any {
	hash := pkgai.UserIDHash(secret, userID)
	rows, err := repoaidisclosure.ListLogsByUserHash(ctx, pool, hash, 1000)
	if err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"feature":        r.FeatureName,
			"modelId":        r.ModelID,
			"provider":       r.Provider,
			"contentHash":    r.ContentHash,
			"optInConfirmed": r.OptInConfirmed,
			"blocked":        r.Blocked,
			"timestamp":      r.Timestamp.UTC().Format(time.RFC3339),
		})
	}
	return out
}
