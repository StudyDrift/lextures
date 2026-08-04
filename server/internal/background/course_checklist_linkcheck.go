package background

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/scheduler"
	"github.com/lextures/lextures/server/internal/service/coursechecklist"
	"github.com/lextures/lextures/server/internal/service/coursechecklist/linkhealth"
)

// RegisterCourseChecklistLinkCheckJobs registers the CC.6 link-health worker and retention sweeper.
func RegisterCourseChecklistLinkCheckJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(coursechecklist.JobTypeChecklistLinkCheck, HandlerFunc(func(ctx context.Context, payload json.RawMessage) error {
		var p struct {
			CourseID string `json:"courseId"`
		}
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		id, err := uuid.Parse(p.CourseID)
		if err != nil {
			return err
		}
		return coursechecklist.RunLinkCheckJob(ctx, pool, id)
	}))

	r.Register(scheduler.JobTypeCourseChecklistLinkHealthRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)
		n, err := linkhealth.SweepOlderThan(ctx, pool, cutoff)
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.course_checklist_link_health_retention", "deleted", n)
		}
		return nil
	}))
}
