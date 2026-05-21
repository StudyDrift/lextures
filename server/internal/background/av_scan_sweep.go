package background

import (
	"context"
	"log/slog"

	"github.com/lextures/lextures/server/internal/workers/avscan"
)

func sweepAVScanJobs(ctx context.Context, worker *avscan.Worker) {
	processed, err := worker.ProcessNext(ctx)
	if err != nil {
		slog.Warn("avscan sweep: error", "err", err)
		return
	}
	if processed {
		slog.Debug("avscan sweep: processed one job")
	}
}
