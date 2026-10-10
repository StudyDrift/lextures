package background

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
	"github.com/lextures/lextures/server/internal/service/paymentprovider"
)

func sweepPaymentWebhookJobs(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.FFPaymentsEnabled || pool == nil {
		return
	}
	svcBilling.SweepPaymentWebhookJobs(ctx, pool, paymentprovider.ConfigFrom(cfg), svcBilling.WebhookOptions{
		RevenueShareEnabled:  cfg.FFRevenueShare,
		TaxCollectionEnabled: cfg.FFTaxCollection,
	}, now)
}

// sweepStalePendingPayments resolves Stripe transactions stuck in "pending" (issue #740).
func sweepStalePendingPayments(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.FFPaymentsEnabled || pool == nil {
		return
	}
	pcfg := paymentprovider.ConfigFrom(cfg)
	if pcfg.StripeSecretKey == "" {
		return
	}
	svcBilling.ReconcileStalePendingStripe(ctx, pool, svcBilling.StripeCheckoutSessionLookup(pcfg.StripeSecretKey), now)
}
