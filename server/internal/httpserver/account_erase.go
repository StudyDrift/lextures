package httpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/repos/gdpr"
	platformpeople "github.com/lextures/lextures/server/internal/repos/platformpeople"
	pfrepo "github.com/lextures/lextures/server/internal/repos/productfeedback"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/authservice"
	"github.com/lextures/lextures/server/internal/service/coursereviews"
)

// accountEraseResult is returned after a successful account erasure.
type accountEraseResult struct {
	OrgID uuid.UUID
}

// errAccountAlreadyErased when the user email is already anonymized.
var errAccountAlreadyErased = errors.New("account already erased")

// errAccountSystemProtected when system/service accounts cannot be deleted via self-service.
var errAccountSystemProtected = errors.New("system account cannot be deleted")

// eraseUserAccount deactivates the user, revokes sessions, strips personal data,
// and anonymizes residual content. Shared by admin people-delete and self-service
// DELETE /api/v1/settings/account.
func (d Deps) eraseUserAccount(ctx context.Context, userID uuid.UUID, allowSystem bool) (*accountEraseResult, error) {
	var orgID uuid.UUID
	var email string
	var accountType string
	err := d.Pool.QueryRow(ctx, `
SELECT org_id, email, COALESCE(NULLIF(TRIM(account_type), ''), 'standard')
  FROM "user".users
 WHERE id = $1
`, userID).Scan(&orgID, &email, &accountType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("load user: %w", err)
	}
	if platformpeople.IsErased(email) {
		return nil, errAccountAlreadyErased
	}
	if !allowSystem && accountType == userrepo.AccountTypeSystem {
		return nil, errAccountSystemProtected
	}

	_ = authservice.RevokeAllSessionsForUser(ctx, d.Pool, userID)
	if err := platformpeople.SetActive(ctx, d.Pool, userID, false); err != nil {
		return nil, fmt.Errorf("deactivate user: %w", err)
	}
	if d.LearnerProfileService != nil {
		_ = d.LearnerProfileService.Erase(ctx, userID)
	}
	_ = pfrepo.DeleteByUser(ctx, d.Pool, userID)
	if err := gdpr.AnonymiseUser(ctx, d.Pool, userID); err != nil {
		return nil, fmt.Errorf("anonymise user: %w", err)
	}
	_ = coursereviews.AnonymizeReviewerReviews(ctx, d.Pool, userID)

	return &accountEraseResult{OrgID: orgID}, nil
}
