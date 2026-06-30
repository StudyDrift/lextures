package background

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/licensesvc"
)

func sweepLicenseSeats(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) {
	if !cfg.SeatManagementEnabled || pool == nil {
		return
	}
	svc := licensesvc.New(pool, cfg)
	if n, err := svc.Reconcile(ctx); err != nil {
		slog.Warn("license seat reconcile failed", "err", err)
		return
	} else if n > 0 {
		slog.Info("license seat reconcile updated rows", "count", n)
	}
	if err := svc.SweepUtilizationAlerts(ctx); err != nil {
		slog.Warn("license utilization alert sweep failed", "err", err)
	}
}
