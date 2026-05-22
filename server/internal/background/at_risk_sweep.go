package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/atriskscoring"
)

func sweepAtRiskScores(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, now time.Time) {
	if !cfg.AtRiskAlertsEnabled || pool == nil {
		return
	}
	// Run once per UTC day after 02:00.
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if now.Hour() < 2 {
		return
	}
	var lastRun *time.Time
	_ = pool.QueryRow(ctx, `
SELECT MAX(computed_date)::timestamptz
FROM analytics.at_risk_scores
`).Scan(&lastRun)
	if lastRun != nil && !lastRun.Before(day) {
		return
	}
	svc := atriskscoring.Service{Pool: pool, Config: cfg}
	n, err := svc.RunAllCourses(ctx, day)
	if err != nil {
		slog.Warn("at_risk.sweep_failed", "err", err)
		return
	}
	if n > 0 {
		slog.Info("at_risk.sweep_completed", "enrollments_scored", n, "date", day.Format("2006-01-02"))
	}
}

// RunAtRiskForCourse is exported for admin manual trigger and tests.
func RunAtRiskForCourse(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID uuid.UUID, day time.Time) (int, error) {
	svc := atriskscoring.Service{Pool: pool, Config: cfg}
	return svc.RunForCourse(ctx, courseID, day)
}
