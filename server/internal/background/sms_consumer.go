package background

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/sms"
	"github.com/lextures/lextures/server/internal/smsnotificationqueue"
)

// StartSmsNotificationConsumer runs the RabbitMQ (or in-memory) SMS consumer until ctx is cancelled.
func StartSmsNotificationConsumer(ctx context.Context, queue *smsnotificationqueue.Bus, pool *pgxpool.Pool, cfg config.Config) {
	if queue == nil || pool == nil {
		return
	}
	go func() {
		slog.Info("sms notification queue consumer started", "concurrency", queue.Concurrency())
		if err := queue.Consume(ctx, func(msg smsnotificationqueue.QueueMessage) error {
			return deliverSmsNotification(ctx, pool, cfg, msg)
		}); err != nil && ctx.Err() == nil {
			slog.Error("sms notification queue consumer stopped", "err", err)
		}
	}()
}

func deliverSmsNotification(_ context.Context, _ *pgxpool.Pool, cfg config.Config, msg smsnotificationqueue.QueueMessage) error {
	body := sms.BuildMessage(msg.Title, msg.Body, msg.ActionURL)
	if err := sms.Send(cfg, msg.Phone, body); err != nil {
		return err
	}
	slog.Info("sms_notification.sent", "user_id", msg.UserID, "event_type", msg.EventType)
	return nil
}