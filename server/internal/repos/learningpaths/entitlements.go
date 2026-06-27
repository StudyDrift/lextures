package learningpaths

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	EntitlementTypePathBundle     = "path_bundle"
	EntitlementTypeCoursePurchase = "course_purchase"
	EntitlementStatusActive       = "active"
)

// HasPathEntitlement returns true when the user may enroll in a paid path.
func HasPathEntitlement(ctx context.Context, pool *pgxpool.Pool, userID, pathID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM billing.user_entitlements e
  WHERE e.user_id = $1 AND e.path_id = $2 AND e.status = 'active'
    AND (e.valid_until IS NULL OR e.valid_until > NOW())
)
`, userID, pathID).Scan(&ok)
	return ok, err
}

// PathRequiresPayment is true when bundle_price_cents is set and positive.
func PathRequiresPayment(p *Path) bool {
	return p != nil && p.BundlePriceCents != nil && *p.BundlePriceCents > 0
}
