package background

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
)

func sweepWebhookDeliveries(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	webhooksvc.SweepDueDeliveries(ctx, pool, cfg, now)
}

func sweepWebhookRetention(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	webhooksvc.PurgeRetention(ctx, pool, cfg, now)
}
