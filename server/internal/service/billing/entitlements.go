package billing

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
)

// UserHasCourseAccess is the entitlement check used by enrollment guards (plan 15.3).
func UserHasCourseAccess(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (bool, error) {
	if pool == nil {
		return false, nil
	}
	return repoBilling.HasCourseAccess(ctx, pool, userID, courseID)
}
