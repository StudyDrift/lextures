package transcripts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/transcriptfees"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrOrderNotDraft      = errors.New("order is not editable")
	ErrOrderEmpty         = errors.New("order must have at least one item")
	ErrOrderItemNotFound  = errors.New("order item not found")
	ErrDocumentNotOwned   = errors.New("document not found for user")
)

// Order is a transcript order owned by a student.
type Order struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	OrgID                *uuid.UUID
	Status               OrderStatus
	ConsentID            *uuid.UUID
	TotalAmount          *int
	Currency             *string
	LegacyRequestID      *uuid.UUID
	PaymentStatus        OrderPaymentStatus
	PaymentRef           *string
	WaiverID             *uuid.UUID
	AmountRefunded       int
	FreeAllotmentApplied bool
	CreatedAt            time.Time
	SubmittedAt          *time.Time
	Items                []OrderItem
}

// OrderPaymentStatus is the T05 payment gate state on an order.
type OrderPaymentStatus = transcriptfees.PaymentStatus

// OrderItem is one recipient × document × delivery method.
type OrderItem struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	RecipientID    *uuid.UUID
	DocumentID     *uuid.UUID
	DeliveryMethod DeliveryMethod
	Urgency        OrderUrgency
	FeeAmount      *int
	Status         ItemStatus
	DeliveredAt    *time.Time
	CreatedAt      time.Time
	Recipient      *Recipient
}

// CreateOrderItemInput is one item at order create / add-item time.
type CreateOrderItemInput struct {
	RecipientID    *uuid.UUID
	AdHoc          *AdHocRecipientInput
	DocumentID     *uuid.UUID
	DeliveryMethod DeliveryMethod
	Urgency        OrderUrgency
}

// CreateOrderInput creates a draft order with 1..N items.
type CreateOrderInput struct {
	UserID uuid.UUID
	OrgID  *uuid.UUID
	Items  []CreateOrderItemInput
}

const orderSelectColumns = `
id, user_id, org_id, status, consent_id, total_amount, currency, legacy_request_id,
COALESCE(payment_status, 'unpaid'), payment_ref, waiver_id,
COALESCE(amount_refunded, 0), COALESCE(free_allotment_applied, FALSE),
created_at, submitted_at`

const orderItemSelectColumns = `
id, order_id, recipient_id, document_id, delivery_method, urgency, fee_amount,
status, delivered_at, created_at`

func scanOrder(row pgx.Row, o *Order) error {
	var status string
	var paymentStatus string
	err := row.Scan(
		&o.ID, &o.UserID, &o.OrgID, &status, &o.ConsentID, &o.TotalAmount, &o.Currency,
		&o.LegacyRequestID, &paymentStatus, &o.PaymentRef, &o.WaiverID,
		&o.AmountRefunded, &o.FreeAllotmentApplied, &o.CreatedAt, &o.SubmittedAt,
	)
	if err != nil {
		return err
	}
	o.Status = OrderStatus(status)
	o.PaymentStatus = OrderPaymentStatus(paymentStatus)
	return nil
}

func scanOrderItem(row pgx.Row, it *OrderItem) error {
	var method, urgency, status string
	err := row.Scan(
		&it.ID, &it.OrderID, &it.RecipientID, &it.DocumentID, &method, &urgency,
		&it.FeeAmount, &status, &it.DeliveredAt, &it.CreatedAt,
	)
	if err != nil {
		return err
	}
	it.DeliveryMethod = DeliveryMethod(method)
	it.Urgency = OrderUrgency(urgency)
	it.Status = ItemStatus(status)
	return nil
}

// ValidateItemDelivery checks recipient capabilities and org-enabled methods.
func ValidateItemDelivery(recipient *Recipient, method DeliveryMethod, cfg *Config) error {
	if recipient == nil {
		return ErrRecipientNotFound
	}
	if !recipient.Active {
		return fmt.Errorf("%w: recipient inactive", ErrInvalidDeliveryMethod)
	}
	if !MethodAllowedByCapabilities(method, recipient.Capabilities) {
		return fmt.Errorf("%w: %s not in recipient capabilities", ErrInvalidDeliveryMethod, method)
	}
	enabled := OrgEnabledDeliveryMethods(cfg)
	if !enabled[method] {
		return fmt.Errorf("%w: %s", ErrDeliveryNotOrgEnabled, method)
	}
	return nil
}

// CreateOrder inserts a draft order and its items atomically.
func CreateOrder(ctx context.Context, pool *pgxpool.Pool, cfg *Config, in CreateOrderInput) (*Order, error) {
	if len(in.Items) == 0 {
		return nil, ErrOrderEmpty
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var o Order
	if err := scanOrder(tx.QueryRow(ctx, `
INSERT INTO transcripts.orders (user_id, org_id, status)
VALUES ($1, $2, 'draft')
RETURNING `+orderSelectColumns+`
`, in.UserID, in.OrgID), &o); err != nil {
		return nil, err
	}

	for _, itemIn := range in.Items {
		it, err := insertOrderItemTx(ctx, tx, pool, cfg, o.ID, in.UserID, in.OrgID, itemIn)
		if err != nil {
			return nil, err
		}
		o.Items = append(o.Items, *it)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return GetOrderForUser(ctx, pool, o.ID, in.UserID)
}

func insertOrderItemTx(
	ctx context.Context,
	tx pgx.Tx,
	pool *pgxpool.Pool,
	cfg *Config,
	orderID, userID uuid.UUID,
	orgID *uuid.UUID,
	in CreateOrderItemInput,
) (*OrderItem, error) {
	urgency := in.Urgency
	if urgency == "" {
		urgency = UrgencyStandard
	}
	var recipient *Recipient
	var err error
	switch {
	case in.RecipientID != nil:
		recipient, err = GetRecipient(ctx, pool, *in.RecipientID)
		if err != nil {
			return nil, err
		}
	case in.AdHoc != nil:
		recipient, err = ResolveOrCreateAdHoc(ctx, pool, orgID, *in.AdHoc)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("recipientId or adHocRecipient is required")
	}
	if err := ValidateItemDelivery(recipient, in.DeliveryMethod, cfg); err != nil {
		return nil, err
	}
	if in.DocumentID != nil {
		doc, err := GetDocumentByID(ctx, pool, userID, *in.DocumentID)
		if err != nil {
			return nil, err
		}
		if doc == nil {
			return nil, ErrDocumentNotOwned
		}
	}
	var it OrderItem
	if err := scanOrderItem(tx.QueryRow(ctx, `
INSERT INTO transcripts.order_items (
    order_id, recipient_id, document_id, delivery_method, urgency, status
)
VALUES ($1, $2, $3, $4, $5, 'pending')
RETURNING `+orderItemSelectColumns+`
`, orderID, recipient.ID, in.DocumentID, string(in.DeliveryMethod), string(urgency)), &it); err != nil {
		return nil, err
	}
	it.Recipient = recipient
	return &it, nil
}

// AddOrderItem appends an item to a draft order owned by userID.
func AddOrderItem(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *Config,
	orderID, userID uuid.UUID,
	in CreateOrderItemInput,
) (*Order, error) {
	o, err := GetOrderForUser(ctx, pool, orderID, userID)
	if err != nil {
		return nil, err
	}
	if o.Status != OrderDraft {
		return nil, ErrOrderNotDraft
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := insertOrderItemTx(ctx, tx, pool, cfg, orderID, userID, o.OrgID, in); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return GetOrderForUser(ctx, pool, orderID, userID)
}

// DeleteOrderItem removes an item from a draft order; refuses to leave the order empty.
func DeleteOrderItem(ctx context.Context, pool *pgxpool.Pool, orderID, itemID, userID uuid.UUID) (*Order, error) {
	o, err := GetOrderForUser(ctx, pool, orderID, userID)
	if err != nil {
		return nil, err
	}
	if o.Status != OrderDraft {
		return nil, ErrOrderNotDraft
	}
	if len(o.Items) <= 1 {
		return nil, ErrOrderEmpty
	}
	tag, err := pool.Exec(ctx, `
DELETE FROM transcripts.order_items
WHERE id = $1 AND order_id = $2
`, itemID, orderID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrOrderItemNotFound
	}
	return GetOrderForUser(ctx, pool, orderID, userID)
}

// ListOrdersByUser returns the user's orders newest first (with items).
func ListOrdersByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Order, error) {
	rows, err := pool.Query(ctx, `
SELECT `+orderSelectColumns+`
FROM transcripts.orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 50
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Order
	for rows.Next() {
		var o Order
		if err := scanOrder(rows, &o); err != nil {
			return nil, err
		}
		out = append(out, o)
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

// GetOrderForUser returns an order if owned by userID (404 semantics via ErrOrderNotFound).
func GetOrderForUser(ctx context.Context, pool *pgxpool.Pool, orderID, userID uuid.UUID) (*Order, error) {
	var o Order
	err := scanOrder(pool.QueryRow(ctx, `
SELECT `+orderSelectColumns+`
FROM transcripts.orders
WHERE id = $1 AND user_id = $2
`, orderID, userID), &o)
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

func listOrderItems(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) ([]OrderItem, error) {
	rows, err := pool.Query(ctx, `
SELECT `+orderItemSelectColumns+`
FROM transcripts.order_items
WHERE order_id = $1
ORDER BY created_at ASC
`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrderItem
	for rows.Next() {
		var it OrderItem
		if err := scanOrderItem(rows, &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].RecipientID == nil {
			continue
		}
		rec, err := GetRecipient(ctx, pool, *out[i].RecipientID)
		if err != nil && !errors.Is(err, ErrRecipientNotFound) {
			return nil, err
		}
		if err == nil {
			out[i].Recipient = rec
		}
	}
	return out, nil
}
