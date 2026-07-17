package background

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/transcriptdelivery"
)

// JobTypeTranscriptDelivery is the durable queue type for T06 item delivery.
const JobTypeTranscriptDelivery = "transcript.delivery"

// TranscriptDeliveryPayload identifies one order item to deliver.
type TranscriptDeliveryPayload struct {
	OrderItemID uuid.UUID `json:"orderItemId"`
}

type transcriptDeliveryHandler struct {
	pool *pgxpool.Pool
	cfg  config.Config
}

func (h transcriptDeliveryHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p TranscriptDeliveryPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("transcript.delivery: bad payload: %w", err)
	}
	if p.OrderItemID == uuid.Nil {
		return fmt.Errorf("transcript.delivery: missing orderItemId")
	}
	return transcriptdelivery.DeliverItem(ctx, h.pool, h.cfg, p.OrderItemID)
}

// RegisterTranscriptDeliveryJob registers the transcript.delivery handler.
func RegisterTranscriptDeliveryJob(r *Registry, pool *pgxpool.Pool, cfg config.Config) {
	if r == nil {
		return
	}
	r.Register(JobTypeTranscriptDelivery, transcriptDeliveryHandler{pool: pool, cfg: cfg})
	transcriptsrepo.AfterItemsReady = EnqueueReadyItemsForOrder
}

// EnqueueTranscriptDelivery queues delivery for one ready order item.
func EnqueueTranscriptDelivery(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (uuid.UUID, error) {
	if pool == nil || itemID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("transcript.delivery: missing pool or item")
	}
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     JobTypeTranscriptDelivery,
		Payload:     TranscriptDeliveryPayload{OrderItemID: itemID},
		Priority:    4,
		MaxAttempts: 5,
		UniqueKey:   "transcript-delivery:" + itemID.String(),
	})
}

// EnqueueReadyItemsForOrder enqueues delivery jobs for every ready item on an order.
func EnqueueReadyItemsForOrder(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) {
	if pool == nil || orderID == uuid.Nil {
		return
	}
	ids, err := transcriptsrepo.ListReadyItemIDs(ctx, pool, &orderID)
	if err != nil {
		return
	}
	for _, id := range ids {
		_, _ = EnqueueTranscriptDelivery(ctx, pool, id)
	}
}
