package learnerprofile

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	lprepo "github.com/lextures/lextures/server/internal/repos/learnerprofile"
)

// DefaultRetentionDays is the inactivity window before learner profile data ages out.
const DefaultRetentionDays = 365

// PurgeInactiveProfiles removes profiles for learners inactive beyond retentionDays.
func PurgeInactiveProfiles(ctx context.Context, pool *pgxpool.Pool, retentionDays int) (int64, error) {
	return lprepo.PurgeInactiveProfiles(ctx, pool, retentionDays)
}