package background

import (
	"context"
	"log/slog"

	"github.com/lextures/lextures/server/internal/workers/transcode"
)

// sweepTranscodeJobs processes one queued transcode job per tick.
func sweepTranscodeJobs(ctx context.Context, worker *transcode.Worker) {
	processed, err := worker.ProcessNext(ctx)
	if err != nil {
		slog.Warn("transcode sweep: error", "err", err)
		return
	}
	if processed {
		slog.Debug("transcode sweep: processed one job")
	}
}
