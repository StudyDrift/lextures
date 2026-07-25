package background

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

// StartAdaptiveContentPipelineWorker polls course.adaptive_content_jobs and
// runs AC.3 generation with budget/rate-limit gates (plan AC.4).
// No-op when pool is nil or BackgroundJobsEnabled is false.
func StartAdaptiveContentPipelineWorker(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) {
	if pool == nil || !cfg.BackgroundJobsEnabled {
		return
	}
	concurrency := 2
	if cfg.BackgroundJobsConcurrency > 0 && cfg.BackgroundJobsConcurrency < concurrency {
		concurrency = cfg.BackgroundJobsConcurrency
	}
	deps := acsvc.WorkerDeps{
		Pool:        pool,
		Client:      platformScopedCompleter(pool, cfg),
		Concurrency: concurrency,
		MaxAttempts: acrepo.DefaultJobMaxAttempts,
	}
	// Claim loop: ~1s pickup latency target (AC.4 NFR p95 ≤ 2s).
	go runEvery(ctx, time.Second, func() {
		_ = deps.RunOnce(context.Background())
	})
	// Period token cache reconcile for enabled courses (ai_usage_log is source of truth).
	go runEvery(ctx, time.Hour, func() {
		reconcileACEBudgets(context.Background(), pool)
	})
	// Queue depth gauge refresh.
	go runEvery(ctx, 15*time.Second, func() {
		acsvc.RefreshQueueMetrics(context.Background(), pool)
	})
	slog.Info("adaptive content pipeline worker started", "concurrency", concurrency)
}

func reconcileACEBudgets(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
SELECT id FROM course.courses WHERE adaptive_content_enabled = TRUE
`)
	if err != nil {
		slog.Warn("adaptivecontent: budget reconcile list failed", "err", err)
		return
	}
	defer rows.Close()
	now := time.Now().UTC()
	for rows.Next() {
		var courseID uuid.UUID
		if err := rows.Scan(&courseID); err != nil {
			return
		}
		if _, err := acrepo.ReconcileTokensUsedPeriod(ctx, pool, courseID, now); err != nil {
			slog.Warn("adaptivecontent: budget reconcile failed", "course_id", courseID, "err", err)
		}
	}
}
