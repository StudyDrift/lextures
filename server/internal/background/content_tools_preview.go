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

// RegisterContentToolsPreviewJobs registers the CT.2 nightly preview-state purge.
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
		return nil
	}))
}
