package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/coachingtips"
)

func sweepWeeklyCoachingTips(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	n, err := coachingtips.RunWeeklyBatch(ctx, pool, cfg, platformScopedCompleter(pool, cfg), now)
	if err != nil {
		slog.Warn("coaching tips sweep failed", "err", err)
		return
	}
	if n > 0 {
		slog.Info("coaching tips generated", "count", n)
	}
}
