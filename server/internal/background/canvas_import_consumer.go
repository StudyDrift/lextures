package background

import (
	"context"
	"log/slog"

	"github.com/lextures/lextures/server/internal/canvasimportqueue"
	"github.com/lextures/lextures/server/internal/repos/canvasimportjobs"
)

// CanvasImportProcessor handles a dequeued Canvas import job.
type CanvasImportProcessor interface {
	HandleCanvasImportQueueMessage(ctx context.Context, msg canvasimportjobs.QueueMessage) error
}

// StartCanvasImportConsumer runs the RabbitMQ (or in-memory) consumer until ctx is cancelled.
func StartCanvasImportConsumer(ctx context.Context, queue *canvasimportqueue.Bus, processor CanvasImportProcessor) {
	if queue == nil || processor == nil {
		return
	}
	go func() {
		slog.Info("canvas import queue consumer started")
		if err := queue.Consume(ctx, func(msg canvasimportjobs.QueueMessage) error {
			return processor.HandleCanvasImportQueueMessage(ctx, msg)
		}); err != nil && ctx.Err() == nil {
			slog.Error("canvas import queue consumer stopped", "err", err)
		}
	}()
}
