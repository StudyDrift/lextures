package payments

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Subscription mirrors a recurring billing relationship.
type Subscription struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	Provider         string
	ProviderSubID    string
	PlanID           string
	Status           string
	CurrentPeriodEnd *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// UpsertSubscriptionInput stores or updates a subscription record.
type UpsertSubscriptionInput struct {
	UserID           uuid.UUID
	Provider         string
	ProviderSubID    string
	PlanID           string
	Status           string
	CurrentPeriodEnd *time.Time
}

// Upsert inserts or updates a subscription by provider_sub_id.
func Upsert(ctx context.Context, pool *pgxpool.Pool, in UpsertSubscriptionInput) error {
	status := in.Status
	if status == "" {
		status = SubStatusActive
	}
	_, err := pool.Exec(ctx, `
INSERT INTO payments.subscriptions (
    user_id, provider, provider_sub_id, plan_id, status, current_period_end, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (provider_sub_id) DO UPDATE SET
    status = EXCLUDED.status,
    current_period_end = EXCLUDED.current_period_end,
    updated_at = NOW()
`, in.UserID, in.Provider, in.ProviderSubID, in.PlanID, status, in.CurrentPeriodEnd)
	return err
}

// CancelByProviderSubID marks a subscription canceled.
func CancelByProviderSubID(ctx context.Context, pool *pgxpool.Pool, providerSubID string) error {
	_, err := pool.Exec(ctx, `
UPDATE payments.subscriptions SET status = 'canceled', updated_at = NOW()
WHERE provider_sub_id = $1
`, providerSubID)
	return err
}

// MarkPastDueByUser marks active subscriptions past due for a user.
func MarkPastDueByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	tag, err := pool.Exec(ctx, `
UPDATE payments.subscriptions
SET status = 'past_due', updated_at = NOW()
WHERE user_id = $1 AND status = 'active'
`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
