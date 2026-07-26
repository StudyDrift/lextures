package background

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/scheduler"
)

// RegisterContentToolsPreviewJobs registers CT.2/CT.3 nightly purge jobs.
func RegisterContentToolsPreviewJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeContentToolPreviewPurge, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		n, err := ctrepo.PurgeStalePreviewStates(ctx, pool, 24*time.Hour)
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.content_tool_preview_purge", "deleted", n)
		}
		n2, err := ctrepo.PurgeStaleActionIdempotency(ctx, pool, 24*time.Hour)
		if err != nil {
			return err
		}
		if n2 > 0 {
			slog.Info("scheduled.content_tool_action_idempotency_purge", "deleted", n2)
		}
		return nil
	}))
}
