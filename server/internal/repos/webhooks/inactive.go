package webhooksrepo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InactiveSubscription is an active webhook subscription with no recent delivery
// activity, surfaced by the inactive_integration_alert scheduled job
// (plan 17.4 FR-4).
type InactiveSubscription struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	Label        string
	LastActivity *time.Time
}

// ListInactiveSubscriptions returns active, non-paused subscriptions whose most
// recent delivery is older than the given threshold (or that have never
// delivered), measured against now. These are likely-broken integrations worth
// an operator alert.
func ListInactiveSubscriptions(ctx context.Context, pool *pgxpool.Pool, threshold time.Duration, now time.Time) ([]InactiveSubscription, error) {
	cutoff := now.Add(-threshold)
	rows, err := pool.Query(ctx, `
SELECT s.id, s.org_id, s.label, d.last_activity
FROM integrations.webhook_subscriptions s
LEFT JOIN (
    SELECT subscription_id, max(created_at) AS last_activity
    FROM integrations.webhook_deliveries
    GROUP BY subscription_id
) d ON d.subscription_id = s.id
WHERE s.active = true
  AND s.paused_at IS NULL
  AND (d.last_activity IS NULL OR d.last_activity < $1)
ORDER BY s.created_at
`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InactiveSubscription
	for rows.Next() {
		var s InactiveSubscription
		if err := rows.Scan(&s.ID, &s.OrgID, &s.Label, &s.LastActivity); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
