package background

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/notifevents"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/notifications"
	"github.com/lextures/lextures/server/internal/service/transcriptnotify"
)

// RegisterTranscriptNotifyHooks wires T10 order/delivery notification fan-out.
func RegisterTranscriptNotifyHooks(pool *pgxpool.Pool, cfg config.Config, hub *notifevents.Hub) {
	if pool == nil {
		return
	}
	svc := &transcriptnotify.Service{
		Pool:   pool,
		Config: cfg,
		Email:  &notifications.Service{Pool: pool, Config: cfg},
		Push: &notifications.PushService{
			Pool:   pool,
			Config: cfg,
			SSEHub: hub,
		},
	}
	transcriptsrepo.AfterOrderStatusChange = func(ctx context.Context, p *pgxpool.Pool, order *transcriptsrepo.Order) {
		if order == nil {
			return
		}
		svc.Pool = p
		svc.NotifyOrderStatus(ctx, order)
	}
	transcriptsrepo.AfterDeliveryReceipt = func(ctx context.Context, p *pgxpool.Pool, order *transcriptsrepo.Order, itemID uuid.UUID, status transcriptsrepo.DeliveryAttemptStatus) {
		if order == nil {
			return
		}
		svc.Pool = p
		svc.NotifyDeliveryStatus(ctx, order, itemID, status)
	}
}
