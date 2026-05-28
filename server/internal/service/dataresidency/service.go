// Package dataresidency implements data residency admin permission checks (plan 10.12).
package dataresidency

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// AdminPermission gates all data residency compliance admin actions.
const AdminPermission = "compliance:data-residency:admin:*"

// CheckAdmin returns true when the user holds the data residency admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}
