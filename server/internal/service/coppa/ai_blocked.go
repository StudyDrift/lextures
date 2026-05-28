package coppa

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IsCoppaAIBlocked returns true for coppa_minor accounts without approved parental AI opt-in (plan 10.17 AC-6).
func IsCoppaAIBlocked(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	status, err := GetUserConsentStatus(ctx, pool, userID)
	if err != nil {
		return true, err
	}
	if !status.CoppaMinor {
		return false, nil
	}
	if status.ConsentStatus != ConsentStatusApproved {
		return true, nil
	}
	return !status.AIFeaturesEnabled, nil
}
