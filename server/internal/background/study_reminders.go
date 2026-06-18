package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/studyreminders"
)

func sweepStudyReminders(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.FFStudyReminders || pool == nil {
		return
	}
	svc := &studyreminders.Service{Pool: pool, Config: cfg}
	n, err := svc.RunSweep(ctx, now)
	if err != nil {
		slog.Warn("study_reminders.sweep", "err", err)
		return
	}
	if n > 0 {
		slog.Info("study_reminders.sent", "count", n)
	}
}
