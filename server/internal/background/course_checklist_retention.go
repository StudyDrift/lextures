package background

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/scheduler"
	"github.com/lextures/lextures/server/internal/service/coursechecklist"
)

// RegisterCourseChecklistRetentionJobs registers the CC.2 nightly retention sweeper.
func RegisterCourseChecklistRetentionJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeCourseChecklistRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		snapshots, events, err := coursechecklist.SweepRetention(ctx, pool, time.Now().UTC())
		if err != nil {
			return err
		}
		if snapshots > 0 || events > 0 {
			slog.Info("scheduled.course_checklist_retention",
				"snapshots_deleted", snapshots,
				"events_deleted", events,
			)
		}
		return nil
	}))
}
