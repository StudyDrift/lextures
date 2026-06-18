package background

import (
	"context"
	"log/slog"

	"github.com/lextures/lextures/server/internal/gradingagentqueue"
)

// GradingAgentProcessor handles a dequeued grading-agent batch item.
type GradingAgentProcessor interface {
	HandleGradingAgentQueueMessage(ctx context.Context, msg gradingagentqueue.QueueMessage) error
}

// StartGradingAgentConsumer runs the RabbitMQ (or in-memory) consumer until ctx is cancelled.
func StartGradingAgentConsumer(ctx context.Context, queue *gradingagentqueue.Bus, processor GradingAgentProcessor) {
	if queue == nil || processor == nil {
		return
	}
	go func() {
		slog.Info("grading agent queue consumer started", "concurrency", queue.Concurrency())
		if err := queue.Consume(ctx, func(msg gradingagentqueue.QueueMessage) error {
			return processor.HandleGradingAgentQueueMessage(ctx, msg)
		}); err != nil && ctx.Err() == nil {
			slog.Error("grading agent queue consumer stopped", "err", err)
		}
	}()
}