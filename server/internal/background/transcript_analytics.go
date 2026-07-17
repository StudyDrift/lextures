package background

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/scheduler"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// RegisterTranscriptAnalyticsJobs registers the daily analytics rollup handler (T12).
func RegisterTranscriptAnalyticsJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeTranscriptAnalyticsRollup, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		now := time.Now().UTC()
		// Refresh today and yesterday so late-arriving rows are captured.
		var total int64
		for _, day := range []time.Time{now.AddDate(0, 0, -1), now} {
			n, err := transcripts.RefreshAnalyticsDaily(ctx, pool, nil, day)
			if err != nil {
				telemetry.RecordBusinessEvent("transcripts.analytics.rollup_failed")
				return err
			}
			total += n
		}
		slog.Info("scheduled.transcript_analytics_rollup", "rows", total)
		telemetry.RecordBusinessEvent("transcripts.analytics.rollup_ok")
		return nil
	}))
}
