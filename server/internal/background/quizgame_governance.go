package background

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/scheduler"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// RegisterQuizgameGovernanceJobs registers usage rollup and retention handlers (IQ.11).
func RegisterQuizgameGovernanceJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeQuizgameUsageRollup, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		day := time.Now().UTC().AddDate(0, 0, -1)
		n, err := quizgame.RefreshUsageDaily(ctx, pool, nil, day)
		if err != nil {
			telemetry.RecordBusinessEvent("quizgame.usage.rollup_failed")
			return err
		}
		// Also refresh today so admin dashboards stay current.
		n2, err := quizgame.RefreshUsageDaily(ctx, pool, nil, time.Now().UTC())
		if err != nil {
			telemetry.RecordBusinessEvent("quizgame.usage.rollup_failed")
			return err
		}
		slog.Info("scheduled.quizgame_usage_rollup", "rows", n+n2)
		telemetry.RecordBusinessEvent("quizgame.usage.rollup_ok")
		return nil
	}))

	r.Register(scheduler.JobTypeQuizgameRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		settings, err := quizgame.GetPlatformSettings(ctx, pool)
		if err != nil {
			telemetry.RecordBusinessEvent("quizgame.retention.failed")
			return err
		}
		retentionDays := settings.RetentionDays
		if retentionDays <= 0 {
			retentionDays = quizgame.DefaultRetentionDays
		}
		guestDays := quizgame.GuestRetentionDays(retentionDays)
		now := time.Now().UTC()
		guestCutoff := now.AddDate(0, 0, -guestDays)
		enrolledCutoff := now.AddDate(0, 0, -retentionDays)
		res, err := quizgame.RunRetention(ctx, pool, guestCutoff, enrolledCutoff, 200)
		if err != nil {
			telemetry.RecordBusinessEvent("quizgame.retention.failed")
			return err
		}
		slog.Info("scheduled.quizgame_retention",
			"guestPurged", res.GuestPlayersPurged,
			"responsesAnonymised", res.ResponsesAnonymised,
			"sessionsTouched", res.SessionsTouched,
		)
		telemetry.RecordBusinessEvent("quizgame.retention.ok")
		pending, _ := quizgame.CountPendingReviews(ctx, pool)
		telemetry.RecordBusinessEvent("quizgame.review_queue.depth")
		_ = pending
		return nil
	}))
}
