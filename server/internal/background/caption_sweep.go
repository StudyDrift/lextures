package background

import (
	"context"
	"log/slog"

	"github.com/lextures/lextures/server/internal/workers/captioning"
)

// sweepCaptionJobs processes one queued caption job per tick.
func sweepCaptionJobs(ctx context.Context, worker *captioning.Worker) {
	processed, err := worker.ProcessNext(ctx)
	if err != nil {
		slog.Warn("caption sweep: error", "err", err)
		return
	}
	if processed {
		slog.Debug("caption sweep: processed one job")
	}
}
