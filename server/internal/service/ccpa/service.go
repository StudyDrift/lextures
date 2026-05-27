// Package ccpa implements CCPA/CPRA compliance: Do Not Sell opt-out, GPC signal
// processing, and California privacy rights request handling (plan 10.4;
// Cal. Civ. Code §§ 1798.100–1798.199, CPRA amendments).
package ccpa

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/ccpa"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// AdminPermission gates CCPA admin actions (approve/deny requests).
const AdminPermission = "compliance:ccpa:admin:*"

// RequestDeadlineWarning is the window before due_at at which an escalation is sent (AC-5).
const RequestDeadlineWarning = 5 * 24 * time.Hour

// RequestDeadlineDays is the CPRA statutory response period (§ 1798.130(a)(2)).
const RequestDeadlineDays = 45

var (
	ErrNotFound      = errors.New("ccpa: record not found")
	ErrAlreadyExists = errors.New("ccpa: active request already exists")
	ErrForbidden     = errors.New("ccpa: forbidden")
)

// CheckAdmin returns true when the user holds the compliance:ccpa:admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// GetOptOut returns the current CCPA opt-out state for a user.
func GetOptOut(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (doNotSell bool, limitSensitivePI bool, err error) {
	s, err := repo.GetOptOut(ctx, pool, userID)
	if err != nil {
		return false, false, fmt.Errorf("ccpa: get opt-out: %w", err)
	}
	return s.DoNotSell, s.LimitSensitivePI, nil
}

// SetDoNotSell updates the Do Not Sell or Share flag for a user (CPRA § 1798.120(a)).
// When value is true, the user has opted out of sale/sharing of their personal information.
func SetDoNotSell(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, value bool) error {
	return repo.SetDoNotSell(ctx, pool, userID, value)
}

// SetLimitSensitivePI updates the Limit Use of Sensitive PI flag (CPRA § 1798.121).
func SetLimitSensitivePI(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, value bool) error {
	return repo.SetLimitSensitivePI(ctx, pool, userID, value)
}

// IsDoNotSellActive returns true when the user has opted out of sale/sharing.
// Used for ad-tech/analytics gating (AC-4).
func IsDoNotSellActive(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	s, err := repo.GetOptOut(ctx, pool, userID)
	if err != nil {
		return false, fmt.Errorf("ccpa: check do-not-sell: %w", err)
	}
	return s.DoNotSell, nil
}

// SubmitRequest creates a new CCPA rights request for the authenticated user.
// Returns ErrAlreadyExists when a pending/in-progress request of the same type exists.
func SubmitRequest(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, requesterEmail, requestType string) (uuid.UUID, error) {
	existing, err := repo.ListRequestsForUser(ctx, pool, userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	for _, r := range existing {
		if r.RequestType == requestType && (r.Status == "pending" || r.Status == "verified" || r.Status == "in_progress") {
			return uuid.UUID{}, ErrAlreadyExists
		}
	}
	return repo.InsertRequest(ctx, pool, &userID, requesterEmail, requestType)
}

// GetRequestForUser returns a CCPA request that belongs to the given user, or ErrNotFound.
func GetRequestForUser(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID) (*repo.CCPARequest, error) {
	r, err := repo.GetRequest(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if r == nil || r.UserID == nil || *r.UserID != userID {
		return nil, ErrNotFound
	}
	return r, nil
}

// ListRequestsForUser returns all CCPA requests submitted by the user.
func ListRequestsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]repo.CCPARequest, error) {
	return repo.ListRequestsForUser(ctx, pool, userID)
}

// ListPendingRequests returns all pending/in-progress requests for admin review.
func ListPendingRequests(ctx context.Context, pool *pgxpool.Pool) ([]repo.CCPARequest, error) {
	return repo.ListPendingRequests(ctx, pool, 500)
}

// ApproveRequest transitions a request to completed with an optional response payload.
func ApproveRequest(ctx context.Context, pool *pgxpool.Pool, id, adminID uuid.UUID) error {
	r, err := repo.GetRequest(ctx, pool, id)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrNotFound
	}
	payload := buildApprovalPayload(r)
	return repo.UpdateRequestStatus(ctx, pool, id, adminID, "completed", &payload)
}

// DenyRequest marks a request as denied with a reason.
func DenyRequest(ctx context.Context, pool *pgxpool.Pool, id, adminID uuid.UUID, reason string) error {
	r, err := repo.GetRequest(ctx, pool, id)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrNotFound
	}
	return repo.UpdateRequestStatus(ctx, pool, id, adminID, "denied", &reason)
}

// CountOverdueRequests returns how many requests are past their 45-day deadline.
func CountOverdueRequests(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	return repo.CountOverdueRequests(ctx, pool)
}

// ListRequestsDueSoon returns requests expiring within RequestDeadlineWarning (5 days).
func ListRequestsDueSoon(ctx context.Context, pool *pgxpool.Pool) ([]repo.CCPARequest, error) {
	return repo.ListRequestsDueSoon(ctx, pool, RequestDeadlineWarning)
}

// PICategories returns the categories of personal information Lextures collects,
// as required by CPRA § 1798.100(a) for the privacy notice.
func PICategories() []PICategory {
	return []PICategory{
		{
			Category:    "Identifiers",
			Examples:    []string{"name", "email address", "account ID"},
			Purpose:     "Account creation, authentication, and service delivery",
			ThirdParties: []string{"AWS (infrastructure)", "SendGrid (email delivery)"},
		},
		{
			Category:    "Internet or other electronic network activity",
			Examples:    []string{"browsing history within the platform", "search queries", "page views"},
			Purpose:     "Adaptive learning, platform analytics",
			ThirdParties: []string{"AWS (infrastructure)"},
		},
		{
			Category:    "Geolocation data",
			Examples:    []string{"IP-derived country/region"},
			Purpose:     "Fraud prevention, compliance (FERPA, COPPA)"},
		{
			Category:    "Inferences drawn from personal information",
			Examples:    []string{"learning style", "knowledge gaps", "at-risk indicators"},
			Purpose:     "AI-assisted tutoring and adaptive recommendations",
			ThirdParties: []string{"OpenRouter (AI model routing)"},
			Sensitive:   true,
		},
		{
			Category:    "Education records",
			Examples:    []string{"grades", "course enrollment", "assignment submissions"},
			Purpose:     "Course delivery, gradebook, LMS functionality",
			ThirdParties: []string{"AWS (infrastructure)"},
		},
	}
}

// PICategory represents one category of personal information for the privacy notice.
type PICategory struct {
	Category     string   `json:"category"`
	Examples     []string `json:"examples"`
	Purpose      string   `json:"purpose"`
	ThirdParties []string `json:"thirdParties,omitempty"`
	Sensitive    bool     `json:"sensitive,omitempty"`
}

func buildApprovalPayload(r *repo.CCPARequest) string {
	switch r.RequestType {
	case "delete":
		return `{"status":"approved","action":"erasure_scheduled","message":"Your personal information will be deleted within 45 days as required by CPRA § 1798.105."}`
	case "correct":
		return `{"status":"approved","action":"correction_pending","message":"Please contact privacy@lextures.com with the corrections you would like made to your personal information."}`
	case "limit_sensitive":
		return `{"status":"approved","action":"limit_applied","message":"Use of your sensitive personal information has been limited to service delivery purposes only."}`
	default:
		return `{"status":"approved","action":"export_available","message":"Your personal information export is being prepared. You will be notified when it is ready."}`
	}
}
