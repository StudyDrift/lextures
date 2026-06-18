package background

import (
	"context"
	"log/slog"

	"github.com/lextures/lextures/server/internal/canvassubmissionsyncqueue"
)

// CanvasSubmissionSyncProcessor handles a dequeued Canvas submission sync job.
type CanvasSubmissionSyncProcessor interface {
	HandleCanvasSubmissionSyncQueueMessage(ctx context.Context, msg canvassubmissionsyncqueue.QueueMessage) error
}

// StartCanvasSubmissionSyncConsumer runs the RabbitMQ (or in-memory) consumer until ctx is cancelled.
func StartCanvasSubmissionSyncConsumer(ctx context.Context, queue *canvassubmissionsyncqueue.Bus, processor CanvasSubmissionSyncProcessor) {
	if queue == nil || processor == nil {
		return
	}
	go func() {
		slog.Info("canvas submission sync queue consumer started", "concurrency", queue.Concurrency())
		if err := queue.Consume(ctx, func(msg canvassubmissionsyncqueue.QueueMessage) error {
			return processor.HandleCanvasSubmissionSyncQueueMessage(ctx, msg)
		}); err != nil && ctx.Err() == nil {
			slog.Error("canvas submission sync queue consumer stopped", "err", err)
		}
	}()
}