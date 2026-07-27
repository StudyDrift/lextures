package background

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/scheduler"
	ctanalytics "github.com/lextures/lextures/server/internal/service/contenttools/analytics"
)

// RegisterContentToolsAnalyticsJobs registers CT.7 nightly rollup + summary rebuild workers.
func RegisterContentToolsAnalyticsJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeContentToolDailyRollups, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		day := time.Now().UTC().AddDate(0, 0, -1)
		if err := ctrepo.ComputeDailyRollupsFromSummaries(ctx, pool, day); err != nil {
			return err
		}
		slog.Info("scheduled.content_tool_daily_rollups", "day", day.Format("2006-01-02"))
		return nil
	}))
	r.Register(scheduler.JobTypeContentToolSummaryRebuild, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		n, err := ctanalytics.RebuildSummaries(ctx, pool, "", 5000)
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.content_tool_summary_rebuild", "rebuilt", n)
		}
		return nil
	}))
}
