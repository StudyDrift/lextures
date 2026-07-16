package background

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/plagiarism"
)

func sweepOriginalityScans(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) {
	if !cfg.FFPlagiarismChecks || !cfg.OriginalityDetectionEnabled {
		return
	}
	svc := &plagiarism.Service{
		Pool:         pool,
		Config:       cfg,
		FilesRoot:    cfg.CourseFilesRoot,
		AI:           platformScopedCompleter(pool, cfg),
		StubExternal: cfg.OriginalityStubExternal,
	}
	if n, err := svc.SweepEnqueue(ctx); err != nil {
		slog.Warn("originality enqueue sweep failed", "err", err)
	} else if n > 0 {
		slog.Debug("originality enqueue sweep", "count", n)
	}
	for {
		done, err := svc.ProcessNext(ctx)
		if err != nil {
			slog.Warn("originality scan failed", "err", err)
			break
		}
		if !done {
			break
		}
	}
}
