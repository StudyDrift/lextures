package transcripts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/transcriptorder"
	"github.com/lextures/lextures/server/internal/telemetry"
)

var (
	ErrIllegalOrderTransition = transcriptorder.ErrIllegalTransition
	ErrTransitionReasonRequired = transcriptorder.ErrReasonRequired
)

// OrderEvent is one immutable state transition.
type OrderEvent struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ItemID    *uuid.UUID
	FromState *string
	ToState   string
	ActorID   *uuid.UUID
	Reason    *string
	CreatedAt time.Time
}

// TransitionInput applies a registrar/student action to an order.
type TransitionInput struct {
	OrderID uuid.UUID
	ActorID *uuid.UUID
	Action  transcriptorder.Action
	Reason  string
	// AutoApproval / gate overrides used for release/submit resolution.
	AutoApproval bool
}

// AdminOrderListFilter filters the registrar fulfillment queue.
type AdminOrderListFilter struct {
	Status   string // empty = fulfillment queue defaults
	Hold     *bool  // true = on_hold only; false = not on_hold; nil = any
	Query    string
	OrgID    *uuid.UUID
	Limit    int
	Offset   int
}

// AdminOrderRow is a queue row with student email and hold summary.
type AdminOrderRow struct {
	Order
	UserEmail       string
	ActiveHoldCount int
	OldestHoldMsg   *string
	Events          []OrderEvent
}

// GateContextForOrder builds consent/payment/hold gates for forward transitions.
// Payment defaults satisfied until T05 wires real checks.
func GateContextForOrder(
	ctx context.Context,
	pool *pgxpool.Pool,
	o *Order,
	autoApproval bool,
) (transcriptorder.GateContext, error) {
	blocked, err := HasBlockingHold(ctx, pool, o.UserID, o.OrgID)
	if err != nil {
		return transcriptorder.GateContext{}, err
	}
	cfg, err := GetConfig(ctx, pool)
	if err != nil {
		return transcriptorder.GateContext{}, err
	}
	consentOK, err := ConsentSatisfiedForOrder(ctx, pool, cfg, o)
	if err != nil {
		return transcriptorder.GateContext{}, err
	}
	return transcriptorder.GateContext{
		ConsentSatisfied: consentOK,
		PaymentSatisfied: true, // T05
		HasBlockingHold:  blocked,
		AutoApproval:     autoApproval,
	}, nil
}

// SubmitOrder advances a draft through hold/consent/payment gates into the lifecycle.
func SubmitOrder(ctx context.Context, pool *pgxpool.Pool, cfg *Config, orderID, userID uuid.UUID) (*Order, error) {
	o, err := GetOrderForUser(ctx, pool, orderID, userID)
	if err != nil {
		return nil, err
	}
	if o.Status != OrderDraft {
		return nil, ErrOrderNotDraft
	}
	if len(o.Items) == 0 {
		return nil, ErrOrderEmpty
	}
	for _, it := range o.Items {
		if it.Recipient == nil {
			return nil, ErrRecipientNotFound
		}
		if err := ValidateItemDelivery(it.Recipient, it.DeliveryMethod, cfg); err != nil {
			return nil, err
		}
	}
	auto := cfg != nil && cfg.AutoApprovalEnabled
	gates, err := GateContextForOrder(ctx, pool, o, auto)
	if err != nil {
		return nil, err
	}
	target := transcriptorder.ResolveSubmitTarget(gates)
	reason := "submitted"
	if gates.HasBlockingHold {
		reason = "blocked by active hold"
	} else if !gates.ConsentSatisfied {
		reason = "awaiting FERPA consent"
	} else if auto && target == transcriptorder.OrderProcessing {
		reason = "auto-approved"
	}
	actor := userID
	if _, err := transitionOrderTx(ctx, pool, transitionParams{
		OrderID: orderID,
		ActorID: &actor,
		From:    OrderDraft,
		To:      OrderStatus(target),
		Reason:  &reason,
		MarkSubmitted: true,
		MarkItemsReady: target == transcriptorder.OrderProcessing,
	}); err != nil {
		return nil, err
	}
	if OrderIsSelfDisclosureOnly(o) {
		_ = LogSelfDisclosureIfNeeded(ctx, pool, o, userID)
	}
	return GetOrderForUser(ctx, pool, orderID, userID)
}

// TransitionOrder applies a registrar action (approve/reject/cancel/complete/hold/release).
func TransitionOrder(ctx context.Context, pool *pgxpool.Pool, cfg *Config, in TransitionInput) (*Order, error) {
	o, err := GetOrderByID(ctx, pool, in.OrderID)
	if err != nil {
		return nil, err
	}
	from := o.Status
	var to OrderStatus
	var reason *string
	if strings.TrimSpace(in.Reason) != "" {
		r := strings.TrimSpace(in.Reason)
		reason = &r
	}

	switch in.Action {
	case transcriptorder.ActionRelease:
		gates, gerr := GateContextForOrder(ctx, pool, o, in.AutoApproval || (cfg != nil && cfg.AutoApprovalEnabled))
		if gerr != nil {
			return nil, gerr
		}
		if from != OrderStatus(transcriptorder.OrderOnHold) {
			return nil, fmt.Errorf("%w: release requires on_hold", ErrIllegalOrderTransition)
		}
		if gates.HasBlockingHold {
			return nil, fmt.Errorf("%w: active holds remain", ErrIllegalOrderTransition)
		}
		to = OrderStatus(transcriptorder.ResolveReleaseTarget(gates))
		if reason == nil {
			r := "hold released"
			reason = &r
		}
	case transcriptorder.ActionHold:
		target, aerr := transcriptorder.TargetForAction(
			transcriptorder.OrderStatus(from), in.Action, in.Reason,
		)
		if aerr != nil {
			return nil, aerr
		}
		to = OrderStatus(target)
		if reason == nil {
			r := "placed on hold"
			reason = &r
		}
	default:
		target, aerr := transcriptorder.TargetForAction(
			transcriptorder.OrderStatus(from), in.Action, in.Reason,
		)
		if aerr != nil {
			return nil, aerr
		}
		to = OrderStatus(target)
	}

	if err := transcriptorder.ValidateOrderTransition(
		transcriptorder.OrderStatus(from),
		transcriptorder.OrderStatus(to),
	); err != nil {
		return nil, err
	}

	// Re-verify holds and consent before any forward path that could issue.
	if to == OrderStatus(transcriptorder.OrderProcessing) || to == OrderStatus(transcriptorder.OrderCompleted) ||
		to == OrderStatus(transcriptorder.OrderInReview) {
		gates, gerr := GateContextForOrder(ctx, pool, o, cfg != nil && cfg.AutoApprovalEnabled)
		if gerr != nil {
			return nil, gerr
		}
		if gates.HasBlockingHold {
			to = OrderStatus(transcriptorder.OrderOnHold)
			r := "blocked by active hold"
			reason = &r
		} else if !gates.ConsentSatisfied {
			to = OrderStatus(transcriptorder.OrderPendingConsent)
			r := "blocked by missing or revoked consent"
			reason = &r
			telemetry.RecordBusinessEvent("transcript_consent_gate_blocked")
		}
		if err := transcriptorder.ValidateOrderTransition(
			transcriptorder.OrderStatus(from),
			transcriptorder.OrderStatus(to),
		); err != nil {
			return nil, err
		}
	}

	markReady := in.Action == transcriptorder.ActionApprove ||
		(in.Action == transcriptorder.ActionRelease && to == OrderStatus(transcriptorder.OrderProcessing))
	cancelItems := in.Action == transcriptorder.ActionReject || in.Action == transcriptorder.ActionCancel

	if _, err := transitionOrderTx(ctx, pool, transitionParams{
		OrderID:        in.OrderID,
		ActorID:        in.ActorID,
		From:           from,
		To:             to,
		Reason:         reason,
		MarkItemsReady: markReady,
		CancelItems:    cancelItems,
	}); err != nil {
		return nil, err
	}
	return GetOrderByID(ctx, pool, in.OrderID)
}

type transitionParams struct {
	OrderID        uuid.UUID
	ActorID        *uuid.UUID
	From           OrderStatus
	To             OrderStatus
	Reason         *string
	MarkSubmitted  bool
	MarkItemsReady bool
	CancelItems    bool
}

func transitionOrderTx(ctx context.Context, pool *pgxpool.Pool, p transitionParams) (*Order, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var o Order
	err = scanOrder(tx.QueryRow(ctx, `
SELECT `+orderSelectColumns+`
FROM transcripts.orders
WHERE id = $1
FOR UPDATE
`, p.OrderID), &o)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Status != p.From {
		return nil, fmt.Errorf("%w: concurrent change (was %s, expected %s)", ErrIllegalOrderTransition, o.Status, p.From)
	}
	if err := transcriptorder.ValidateOrderTransition(
		transcriptorder.OrderStatus(p.From),
		transcriptorder.OrderStatus(p.To),
	); err != nil {
		return nil, err
	}

	if p.MarkSubmitted {
		_, err = tx.Exec(ctx, `
UPDATE transcripts.orders
SET status = $2, submitted_at = COALESCE(submitted_at, NOW())
WHERE id = $1
`, p.OrderID, string(p.To))
	} else {
		_, err = tx.Exec(ctx, `
UPDATE transcripts.orders
SET status = $2
WHERE id = $1
`, p.OrderID, string(p.To))
	}
	if err != nil {
		return nil, err
	}

	fromStr := string(p.From)
	if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, from_state, to_state, actor_id, reason)
VALUES ($1, $2, $3, $4, $5)
`, p.OrderID, fromStr, string(p.To), p.ActorID, p.Reason); err != nil {
		return nil, err
	}

	if p.MarkItemsReady {
		if err := markItemsStatusTx(ctx, tx, p.OrderID, p.ActorID, ItemPending, ItemReady, "ready for delivery"); err != nil {
			return nil, err
		}
	}
	if p.CancelItems {
		if err := cancelOpenItemsTx(ctx, tx, p.OrderID, p.ActorID, p.Reason); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return GetOrderByID(ctx, pool, p.OrderID)
}

func markItemsStatusTx(
	ctx context.Context,
	tx pgx.Tx,
	orderID uuid.UUID,
	actorID *uuid.UUID,
	from, to ItemStatus,
	reason string,
) error {
	rows, err := tx.Query(ctx, `
SELECT id, status FROM transcripts.order_items WHERE order_id = $1 AND status = $2
`, orderID, string(from))
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		var st string
		if err := rows.Scan(&id, &st); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if err := transcriptorder.ValidateItemTransition(
			transcriptorder.ItemStatus(from),
			transcriptorder.ItemStatus(to),
		); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
UPDATE transcripts.order_items SET status = $2 WHERE id = $1
`, id, string(to)); err != nil {
			return err
		}
		fromStr := string(from)
		r := reason
		if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, item_id, from_state, to_state, actor_id, reason)
VALUES ($1, $2, $3, $4, $5, $6)
`, orderID, id, fromStr, string(to), actorID, r); err != nil {
			return err
		}
	}
	return nil
}

func cancelOpenItemsTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, actorID *uuid.UUID, reason *string) error {
	rows, err := tx.Query(ctx, `
SELECT id, status FROM transcripts.order_items
WHERE order_id = $1 AND status IN ('pending', 'ready', 'delivering')
`, orderID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		id uuid.UUID
		st ItemStatus
	}
	var list []row
	for rows.Next() {
		var r row
		var st string
		if err := rows.Scan(&r.id, &st); err != nil {
			return err
		}
		r.st = ItemStatus(st)
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rtext := "canceled"
	if reason != nil && strings.TrimSpace(*reason) != "" {
		rtext = strings.TrimSpace(*reason)
	}
	for _, it := range list {
		to := ItemCanceled
		if err := transcriptorder.ValidateItemTransition(
			transcriptorder.ItemStatus(it.st),
			transcriptorder.ItemStatus(to),
		); err != nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
UPDATE transcripts.order_items SET status = $2 WHERE id = $1
`, it.id, string(to)); err != nil {
			return err
		}
		fromStr := string(it.st)
		if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, item_id, from_state, to_state, actor_id, reason)
VALUES ($1, $2, $3, $4, $5, $6)
`, orderID, it.id, fromStr, string(to), actorID, rtext); err != nil {
			return err
		}
	}
	return nil
}

// GetOrderByID loads an order without user ownership check (admin).
func GetOrderByID(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) (*Order, error) {
	var o Order
	err := scanOrder(pool.QueryRow(ctx, `
SELECT `+orderSelectColumns+`
FROM transcripts.orders
WHERE id = $1
`, orderID), &o)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := listOrderItems(ctx, pool, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return &o, nil
}

// ListOrderEvents returns transition history oldest-first.
func ListOrderEvents(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) ([]OrderEvent, error) {
	rows, err := pool.Query(ctx, `
SELECT id, order_id, item_id, from_state, to_state, actor_id, reason, created_at
FROM transcripts.order_events
WHERE order_id = $1
ORDER BY created_at ASC
`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderEvent
	for rows.Next() {
		var e OrderEvent
		if err := rows.Scan(
			&e.ID, &e.OrderID, &e.ItemID, &e.FromState, &e.ToState, &e.ActorID, &e.Reason, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListAdminOrders returns the registrar fulfillment queue.
func ListAdminOrders(ctx context.Context, pool *pgxpool.Pool, f AdminOrderListFilter) ([]AdminOrderRow, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	statusFilter := strings.TrimSpace(strings.ToLower(f.Status))
	q := strings.TrimSpace(f.Query)

	rows, err := pool.Query(ctx, `
SELECT o.id, o.user_id, o.org_id, o.status, o.consent_id, o.total_amount, o.currency, o.legacy_request_id,
       o.created_at, o.submitted_at,
       u.email,
       COALESCE((
         SELECT COUNT(*)::int FROM transcripts.holds h
         WHERE h.user_id = o.user_id AND h.released_at IS NULL
           AND (o.org_id IS NULL OR h.org_id IS NULL OR h.org_id = o.org_id)
       ), 0) AS hold_count,
       (
         SELECT h.student_message FROM transcripts.holds h
         WHERE h.user_id = o.user_id AND h.released_at IS NULL
           AND (o.org_id IS NULL OR h.org_id IS NULL OR h.org_id = o.org_id)
         ORDER BY h.placed_at ASC
         LIMIT 1
       ) AS hold_msg
FROM transcripts.orders o
JOIN "user".users u ON u.id = o.user_id
WHERE o.status <> 'draft'
  AND ($1::text = '' OR o.status = $1)
  AND (
    $2::boolean IS NULL
    OR ($2::boolean = TRUE AND o.status = 'on_hold')
    OR ($2::boolean = FALSE AND o.status <> 'on_hold')
  )
  AND ($3::uuid IS NULL OR o.org_id IS NULL OR o.org_id = $3)
  AND (
    $4::text = ''
    OR u.email ILIKE '%' || $4 || '%'
    OR o.id::text ILIKE '%' || $4 || '%'
  )
ORDER BY
  CASE o.status
    WHEN 'on_hold' THEN 0
    WHEN 'in_review' THEN 1
    WHEN 'pending_consent' THEN 2
    WHEN 'pending_payment' THEN 3
    WHEN 'processing' THEN 4
    ELSE 5
  END,
  COALESCE(o.submitted_at, o.created_at) ASC
LIMIT $5 OFFSET $6
`, statusFilter, f.Hold, f.OrgID, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AdminOrderRow
	for rows.Next() {
		var row AdminOrderRow
		var status string
		var holdMsg *string
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.OrgID, &status, &row.ConsentID, &row.TotalAmount, &row.Currency,
			&row.LegacyRequestID, &row.CreatedAt, &row.SubmittedAt,
			&row.UserEmail, &row.ActiveHoldCount, &holdMsg,
		); err != nil {
			return nil, err
		}
		row.Status = OrderStatus(status)
		row.OldestHoldMsg = holdMsg
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		items, err := listOrderItems(ctx, pool, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

// ReevaluateOrdersAfterHoldChange moves orders for a user onto/off hold after hold place/release.
func ReevaluateOrdersAfterHoldChange(ctx context.Context, pool *pgxpool.Pool, cfg *Config, userID uuid.UUID, orgID *uuid.UUID, actorID *uuid.UUID) error {
	blocked, err := HasBlockingHold(ctx, pool, userID, orgID)
	if err != nil {
		return err
	}
	auto := cfg != nil && cfg.AutoApprovalEnabled

	if blocked {
		rows, err := pool.Query(ctx, `
SELECT id FROM transcripts.orders
WHERE user_id = $1
  AND status IN ('in_review', 'processing', 'pending_consent', 'pending_payment')
  AND ($2::uuid IS NULL OR org_id IS NULL OR org_id = $2)
`, userID, orgID)
		if err != nil {
			return err
		}
		defer rows.Close()
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for _, id := range ids {
			_, _ = TransitionOrder(ctx, pool, cfg, TransitionInput{
				OrderID: id,
				ActorID: actorID,
				Action:  transcriptorder.ActionHold,
				Reason:  "active hold placed",
			})
		}
		return nil
	}

	rows, err := pool.Query(ctx, `
SELECT id FROM transcripts.orders
WHERE user_id = $1
  AND status = 'on_hold'
  AND ($2::uuid IS NULL OR org_id IS NULL OR org_id = $2)
`, userID, orgID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		_, _ = TransitionOrder(ctx, pool, cfg, TransitionInput{
			OrderID:      id,
			ActorID:      actorID,
			Action:       transcriptorder.ActionRelease,
			Reason:       "hold released",
			AutoApproval: auto,
		})
	}
	return nil
}

// RejectionReasonFromEvents returns the latest reject reason for student display.
func RejectionReasonFromEvents(events []OrderEvent) *string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].ToState == string(transcriptorder.OrderRejected) && events[i].Reason != nil {
			return events[i].Reason
		}
	}
	return nil
}
