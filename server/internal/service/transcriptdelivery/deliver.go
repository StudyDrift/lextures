package transcriptdelivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// DeliverItem runs the release guard + adapter for one order item (idempotent).
func DeliverItem(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, itemID uuid.UUID) error {
	env := &Env{Pool: pool, Cfg: cfg}
	dc, err := transcriptsrepo.LoadDeliveryItemContext(ctx, pool, itemID)
	if err != nil {
		return err
	}

	switch dc.Item.Status {
	case transcriptsrepo.ItemDelivered:
		return nil
	case transcriptsrepo.ItemCanceled, transcriptsrepo.ItemFailed:
		return fmt.Errorf("%w: item status %s", transcriptsrepo.ErrItemNotReady, dc.Item.Status)
	case transcriptsrepo.ItemReady, transcriptsrepo.ItemDelivering:
		// proceed
	default:
		return fmt.Errorf("%w: item status %s", transcriptsrepo.ErrItemNotReady, dc.Item.Status)
	}

	if dc.Document == nil {
		return transcriptsrepo.ErrDocumentRequired
	}
	if !transcriptsrepo.VerifyDocumentHash(dc.Document) {
		return fmt.Errorf("document integrity check failed")
	}

	guard, err := ReleaseGuard(ctx, pool, &dc.Order)
	if err != nil {
		return err
	}
	if !guard.OK {
		detail := guard.Reason
		if guard.OnHold {
			_ = transcriptsrepo.AbortOrderToHold(ctx, pool, dc.Order.ID, detail)
		}
		telemetry.RecordBusinessEvent("transcript.delivery.guard_denied")
		return fmt.Errorf("%w: %s", transcriptsrepo.ErrReleaseGuardDenied, detail)
	}

	tcfg, err := transcriptsrepo.GetConfig(ctx, pool)
	if err != nil {
		return err
	}
	adapter, err := SelectAdapter(dc.Item.DeliveryMethod, tcfg.DeliveryV2)
	if err != nil {
		return err
	}

	// Crash recovery when already claimed: adapter succeeded but item not marked.
	if dc.Item.Status == transcriptsrepo.ItemDelivering {
		if prior, lerr := transcriptsrepo.ListDeliveryAttemptsForItem(ctx, pool, itemID); lerr == nil && len(prior) > 0 {
			latest := prior[len(prior)-1]
			if latest.Status == transcriptsrepo.AttemptDelivered ||
				latest.Status == transcriptsrepo.AttemptOpened ||
				latest.Status == transcriptsrepo.AttemptSent {
				_ = transcriptsrepo.MarkItemDelivered(ctx, pool, itemID)
				_ = transcriptsrepo.MaybeCompleteOrder(ctx, pool, dc.Order.ID)
				return nil
			}
		}
	}

	claimed, err := transcriptsrepo.ClaimItemForDelivery(ctx, pool, itemID)
	if err != nil {
		return err
	}
	if !claimed {
		return fmt.Errorf("%w: could not claim item", transcriptsrepo.ErrItemNotReady)
	}

	attempt, err := beginAttempt(ctx, pool, itemID, dc.Item.DeliveryMethod)
	if err != nil {
		return err
	}
	if attempt.Status == transcriptsrepo.AttemptDelivered || attempt.Status == transcriptsrepo.AttemptOpened {
		_ = transcriptsrepo.MarkItemDelivered(ctx, pool, itemID)
		_ = transcriptsrepo.MaybeCompleteOrder(ctx, pool, dc.Order.ID)
		return nil
	}
	attemptNo := attempt.AttemptNo

	telemetry.RecordBusinessEvent("transcript.delivery.attempted")
	result, derr := adapter.Deliver(ctx, env, dc, attempt)
	if derr != nil {
		detail := derr.Error()
		// Never log document bytes — detail is adapter error only.
		slog.Warn("transcript.delivery.failed",
			"order_item_id", itemID.String(),
			"adapter", string(dc.Item.DeliveryMethod),
			"attempt", attemptNo,
			"err", detail,
		)
		status := transcriptsrepo.AttemptFailed
		if result.Status != "" {
			status = result.Status
		}
		_, _ = transcriptsrepo.UpdateDeliveryAttemptStatus(ctx, pool, attempt.ID, status, result.ResponseCode, &detail)
		if errors.Is(derr, ErrTransient) {
			_ = transcriptsrepo.RevertItemToReady(ctx, pool, itemID)
			telemetry.RecordBusinessEvent("transcript.delivery.failed")
			return derr
		}
		_ = transcriptsrepo.MarkItemFailed(ctx, pool, itemID, detail)
		telemetry.RecordBusinessEvent("transcript.delivery.failed")
		if order, oerr := transcriptsrepo.GetOrderByID(ctx, pool, dc.Order.ID); oerr == nil {
			transcriptsrepo.NotifyDeliveryReceipt(ctx, pool, order, itemID, transcriptsrepo.AttemptFailed)
		}
		// Permanent failure: ack the job so it does not retry forever.
		return nil
	}

	detail := result.Detail
	if detail == "" {
		detail = "delivered"
	}
	if result.ShareURL != "" && !strings.Contains(detail, "http") {
		// Keep share URL out of logs; store only a marker in detail.
		detail = strings.TrimSpace(detail + "; share_link_issued")
	}
	status := result.Status
	if status == "" {
		status = transcriptsrepo.AttemptDelivered
	}
	_, _ = transcriptsrepo.UpdateDeliveryAttemptStatus(ctx, pool, attempt.ID, status, result.ResponseCode, &detail)
	if status == transcriptsrepo.AttemptDelivered || status == transcriptsrepo.AttemptSent || status == transcriptsrepo.AttemptOpened {
		if err := transcriptsrepo.MarkItemDelivered(ctx, pool, itemID); err != nil {
			return err
		}
		_ = transcriptsrepo.MaybeCompleteOrder(ctx, pool, dc.Order.ID)
		telemetry.RecordBusinessEvent("transcript.delivery.succeeded")
		if order, oerr := transcriptsrepo.GetOrderByID(ctx, pool, dc.Order.ID); oerr == nil {
			transcriptsrepo.NotifyDeliveryReceipt(ctx, pool, order, itemID, status)
		}
	}
	return nil
}

func beginAttempt(
	ctx context.Context,
	pool *pgxpool.Pool,
	itemID uuid.UUID,
	method transcriptsrepo.DeliveryMethod,
) (*transcriptsrepo.DeliveryAttempt, error) {
	prior, err := transcriptsrepo.ListDeliveryAttemptsForItem(ctx, pool, itemID)
	if err != nil {
		return nil, err
	}
	for i := len(prior) - 1; i >= 0; i-- {
		a := prior[i]
		if a.Status == transcriptsrepo.AttemptQueued || a.Status == transcriptsrepo.AttemptSent {
			return &a, nil
		}
	}
	n, err := transcriptsrepo.NextAttemptNo(ctx, pool, itemID)
	if err != nil {
		return nil, err
	}
	return transcriptsrepo.InsertDeliveryAttempt(ctx, pool, itemID, method, n)
}

// PrepareResend moves a failed/delivered item back to ready for a new attempt.
func PrepareResend(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) error {
	dc, err := transcriptsrepo.LoadDeliveryItemContext(ctx, pool, itemID)
	if err != nil {
		return err
	}
	switch dc.Item.Status {
	case transcriptsrepo.ItemFailed, transcriptsrepo.ItemDelivered, transcriptsrepo.ItemReady:
		// ok
	default:
		return fmt.Errorf("%w: cannot resend from %s", transcriptsrepo.ErrItemNotReady, dc.Item.Status)
	}
	_, err = pool.Exec(ctx, `
UPDATE transcripts.order_items SET status = 'ready', delivered_at = NULL
WHERE id = $1
`, itemID)
	if err != nil {
		return err
	}
	_, _ = pool.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, item_id, from_state, to_state, reason)
VALUES ($1, $2, $3, 'ready', 'resend requested')
`, dc.Order.ID, itemID, string(dc.Item.Status))
	// Ensure order is processing so delivery can complete again.
	_, _ = pool.Exec(ctx, `
UPDATE transcripts.orders SET status = 'processing'
WHERE id = $1 AND status IN ('completed', 'processing')
`, dc.Order.ID)
	return nil
}
