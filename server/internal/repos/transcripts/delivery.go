package transcripts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDeliveryAttemptNotFound = errors.New("delivery attempt not found")
	ErrShareLinkNotFound       = errors.New("share link not found")
	ErrShareLinkExpired        = errors.New("share link expired")
	ErrShareLinkExhausted      = errors.New("share link download limit reached")
	ErrItemNotReady            = errors.New("order item is not ready for delivery")
	ErrDocumentRequired        = errors.New("order item requires a document")
	ErrReleaseGuardDenied      = errors.New("release guard denied delivery")
)

// AfterItemsReady is invoked after order items transition to ready (T06 enqueue hook).
// Wired by the background package to avoid an import cycle.
var AfterItemsReady func(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID)

// NotifyItemsReady calls AfterItemsReady when configured.
func NotifyItemsReady(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) {
	if AfterItemsReady != nil && pool != nil && orderID != uuid.Nil {
		AfterItemsReady(ctx, pool, orderID)
	}
}

// DeliveryAttemptStatus is the receipt/status on a delivery attempt.
type DeliveryAttemptStatus string

const (
	AttemptQueued    DeliveryAttemptStatus = "queued"
	AttemptSent      DeliveryAttemptStatus = "sent"
	AttemptDelivered DeliveryAttemptStatus = "delivered"
	AttemptOpened    DeliveryAttemptStatus = "opened"
	AttemptFailed    DeliveryAttemptStatus = "failed"
)

// DeliveryAttempt is one adapter send for an order item.
type DeliveryAttempt struct {
	ID             uuid.UUID
	OrderItemID    uuid.UUID
	Adapter        DeliveryMethod
	AttemptNo      int
	Status         DeliveryAttemptStatus
	ResponseCode   *int
	Detail         *string
	IdempotencyKey string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ShareLink is a secure recipient download token.
type ShareLink struct {
	ID            uuid.UUID
	OrderItemID   uuid.UUID
	DocumentID    uuid.UUID
	Token         string
	ExpiresAt     time.Time
	MaxDownloads  int
	DownloadCount int
	OpenedAt      *time.Time
	LastIP        *string
	CreatedAt     time.Time
}

// PostalJob is a print/mail fulfillment row.
type PostalJob struct {
	ID          uuid.UUID
	OrderItemID uuid.UUID
	DocumentID  uuid.UUID
	Address     json.RawMessage
	Status      string
	VendorRef   *string
	Detail      *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DeliveryItemContext is everything needed to send one ready item.
type DeliveryItemContext struct {
	Item     OrderItem
	Order    Order
	Document *Document
}

const deliveryAttemptSelect = `
id, order_item_id, adapter, attempt_no, status, response_code, detail,
idempotency_key, created_at, updated_at`

func scanDeliveryAttempt(row pgx.Row, a *DeliveryAttempt) error {
	var adapter, status string
	err := row.Scan(
		&a.ID, &a.OrderItemID, &adapter, &a.AttemptNo, &status, &a.ResponseCode, &a.Detail,
		&a.IdempotencyKey, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return err
	}
	a.Adapter = DeliveryMethod(adapter)
	a.Status = DeliveryAttemptStatus(status)
	return nil
}

// IdempotencyKeyForAttempt builds a stable key for (item, attempt_no).
func IdempotencyKeyForAttempt(itemID uuid.UUID, attemptNo int) string {
	return fmt.Sprintf("transcript-delivery:%s:%d", itemID.String(), attemptNo)
}

// NextAttemptNo returns the next attempt number for an item (1-based).
func NextAttemptNo(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (int, error) {
	var max *int
	err := pool.QueryRow(ctx, `
SELECT MAX(attempt_no) FROM transcripts.delivery_attempts WHERE order_item_id = $1
`, itemID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if max == nil {
		return 1, nil
	}
	return *max + 1, nil
}

// InsertDeliveryAttempt creates a queued attempt row; returns existing row on idempotency conflict.
func InsertDeliveryAttempt(
	ctx context.Context,
	pool *pgxpool.Pool,
	itemID uuid.UUID,
	adapter DeliveryMethod,
	attemptNo int,
) (*DeliveryAttempt, error) {
	key := IdempotencyKeyForAttempt(itemID, attemptNo)
	var a DeliveryAttempt
	err := scanDeliveryAttempt(pool.QueryRow(ctx, `
INSERT INTO transcripts.delivery_attempts (
  order_item_id, adapter, attempt_no, status, idempotency_key
) VALUES ($1, $2, $3, 'queued', $4)
ON CONFLICT (order_item_id, idempotency_key) DO UPDATE
SET updated_at = transcripts.delivery_attempts.updated_at
RETURNING `+deliveryAttemptSelect+`
`, itemID, string(adapter), attemptNo, key), &a)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateDeliveryAttemptStatus sets status/response/detail for an attempt.
func UpdateDeliveryAttemptStatus(
	ctx context.Context,
	pool *pgxpool.Pool,
	attemptID uuid.UUID,
	status DeliveryAttemptStatus,
	responseCode *int,
	detail *string,
) (*DeliveryAttempt, error) {
	var a DeliveryAttempt
	err := scanDeliveryAttempt(pool.QueryRow(ctx, `
UPDATE transcripts.delivery_attempts
SET status = $2, response_code = $3, detail = $4, updated_at = NOW()
WHERE id = $1
RETURNING `+deliveryAttemptSelect+`
`, attemptID, string(status), responseCode, detail), &a)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeliveryAttemptNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListDeliveryAttemptsForItem returns attempts oldest-first (receipt timeline).
func ListDeliveryAttemptsForItem(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) ([]DeliveryAttempt, error) {
	rows, err := pool.Query(ctx, `
SELECT `+deliveryAttemptSelect+`
FROM transcripts.delivery_attempts
WHERE order_item_id = $1
ORDER BY attempt_no ASC, created_at ASC
`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeliveryAttempt
	for rows.Next() {
		var a DeliveryAttempt
		if err := scanDeliveryAttempt(rows, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetDeliveryAttemptByIdempotency returns an attempt if the key already exists.
func GetDeliveryAttemptByIdempotency(
	ctx context.Context,
	pool *pgxpool.Pool,
	itemID uuid.UUID,
	key string,
) (*DeliveryAttempt, error) {
	var a DeliveryAttempt
	err := scanDeliveryAttempt(pool.QueryRow(ctx, `
SELECT `+deliveryAttemptSelect+`
FROM transcripts.delivery_attempts
WHERE order_item_id = $1 AND idempotency_key = $2
`, itemID, key), &a)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeliveryAttemptNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// LoadDeliveryItemContext loads order, item, and document for delivery.
func LoadDeliveryItemContext(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (*DeliveryItemContext, error) {
	var it OrderItem
	err := scanOrderItem(pool.QueryRow(ctx, `
SELECT `+orderItemSelectColumns+`
FROM transcripts.order_items WHERE id = $1
`, itemID), &it)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderItemNotFound
	}
	if err != nil {
		return nil, err
	}
	if it.RecipientID != nil {
		rec, rerr := GetRecipient(ctx, pool, *it.RecipientID)
		if rerr == nil {
			it.Recipient = rec
		}
	}
	o, err := GetOrderByID(ctx, pool, it.OrderID)
	if err != nil {
		return nil, err
	}
	var doc *Document
	if it.DocumentID != nil {
		doc, err = GetDocumentByIDAdmin(ctx, pool, *it.DocumentID)
		if err != nil {
			return nil, err
		}
	}
	return &DeliveryItemContext{Item: it, Order: *o, Document: doc}, nil
}

// ClaimItemForDelivery moves ready → delivering atomically. Returns false if not claimable.
func ClaimItemForDelivery(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (bool, error) {
	var orderID uuid.UUID
	err := pool.QueryRow(ctx, `
UPDATE transcripts.order_items
SET status = 'delivering'
WHERE id = $1 AND status = 'ready'
RETURNING order_id
`, itemID).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already delivering is ok for at-least-once reclaim.
		var st string
		err := pool.QueryRow(ctx, `SELECT status FROM transcripts.order_items WHERE id = $1`, itemID).Scan(&st)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrOrderItemNotFound
		}
		if err != nil {
			return false, err
		}
		return st == string(ItemDelivering), nil
	}
	if err != nil {
		return false, err
	}
	_, _ = pool.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, item_id, from_state, to_state, reason)
VALUES ($1, $2, 'ready', 'delivering', 'delivery claimed')
`, orderID, itemID)
	return true, nil
}

// MarkItemDelivered sets item delivered + delivered_at and records an event.
func MarkItemDelivered(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) error {
	var orderID uuid.UUID
	err := pool.QueryRow(ctx, `
UPDATE transcripts.order_items
SET status = 'delivered', delivered_at = COALESCE(delivered_at, NOW())
WHERE id = $1 AND status IN ('ready', 'delivering')
RETURNING order_id
`, itemID).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		var st string
		err2 := pool.QueryRow(ctx, `SELECT status, order_id FROM transcripts.order_items WHERE id = $1`, itemID).Scan(&st, &orderID)
		if err2 != nil {
			return err2
		}
		if st == string(ItemDelivered) {
			return nil
		}
		return fmt.Errorf("%w: cannot mark delivered from %s", ErrIllegalOrderTransition, st)
	}
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, item_id, from_state, to_state, reason)
VALUES ($1, $2, 'delivering', 'delivered', 'delivered')
`, orderID, itemID)
	return err
}

// MarkItemFailed sets item failed and records an event.
func MarkItemFailed(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, reason string) error {
	var orderID uuid.UUID
	err := pool.QueryRow(ctx, `
UPDATE transcripts.order_items
SET status = 'failed'
WHERE id = $1 AND status IN ('ready', 'delivering')
RETURNING order_id
`, itemID).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	r := strings.TrimSpace(reason)
	if r == "" {
		r = "delivery failed"
	}
	_, err = pool.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, item_id, from_state, to_state, reason)
VALUES ($1, $2, 'delivering', 'failed', $3)
`, orderID, itemID, r)
	return err
}

// RevertItemToReady returns a delivering item to ready (e.g. transient retry before next attempt).
func RevertItemToReady(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE transcripts.order_items SET status = 'ready'
WHERE id = $1 AND status = 'delivering'
`, itemID)
	return err
}

// AbortOrderToHold moves a processing order to on_hold when the release guard fails.
func AbortOrderToHold(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID, reason string) error {
	r := strings.TrimSpace(reason)
	if r == "" {
		r = "release guard denied"
	}
	tag, err := pool.Exec(ctx, `
UPDATE transcripts.orders SET status = 'on_hold'
WHERE id = $1 AND status = 'processing'
`, orderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, err = pool.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, from_state, to_state, reason)
VALUES ($1, 'processing', 'on_hold', $2)
`, orderID, r)
	if err != nil {
		return err
	}
	// Open items return to pending until hold clears / re-approved.
	_, err = pool.Exec(ctx, `
UPDATE transcripts.order_items
SET status = 'pending'
WHERE order_id = $1 AND status IN ('ready', 'delivering')
`, orderID)
	return err
}

// MaybeCompleteOrder marks processing → completed when every item is terminal delivered/canceled.
func MaybeCompleteOrder(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) error {
	var open int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM transcripts.order_items
WHERE order_id = $1 AND status NOT IN ('delivered', 'canceled', 'failed')
`, orderID).Scan(&open)
	if err != nil {
		return err
	}
	if open > 0 {
		return nil
	}
	var failed int
	err = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM transcripts.order_items
WHERE order_id = $1 AND status = 'failed'
`, orderID).Scan(&failed)
	if err != nil {
		return err
	}
	to := "completed"
	if failed > 0 {
		// Keep processing/failed visibility via item states; order completes only when no failures,
		// otherwise leave as processing so registrar can resend.
		return nil
	}
	tag, err := pool.Exec(ctx, `
UPDATE transcripts.orders SET status = $2
WHERE id = $1 AND status = 'processing'
`, orderID, to)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, err = pool.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, from_state, to_state, reason)
VALUES ($1, 'processing', $2, 'all items delivered')
`, orderID, to)
	return err
}

// ListReadyItemIDs returns item ids in ready status for an order (or all when orderID nil).
func ListReadyItemIDs(ctx context.Context, pool *pgxpool.Pool, orderID *uuid.UUID) ([]uuid.UUID, error) {
	var rows pgx.Rows
	var err error
	if orderID != nil {
		rows, err = pool.Query(ctx, `
SELECT id FROM transcripts.order_items WHERE order_id = $1 AND status = 'ready'
`, *orderID)
	} else {
		rows, err = pool.Query(ctx, `
SELECT id FROM transcripts.order_items WHERE status = 'ready' LIMIT 200
`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// GenerateShareToken returns a high-entropy opaque token.
func GenerateShareToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// CreateShareLink inserts a secure download link for an order item.
func CreateShareLink(
	ctx context.Context,
	pool *pgxpool.Pool,
	itemID, documentID uuid.UUID,
	expiresAt time.Time,
	maxDownloads int,
) (*ShareLink, error) {
	if maxDownloads <= 0 {
		maxDownloads = 5
	}
	token, err := GenerateShareToken()
	if err != nil {
		return nil, err
	}
	var sl ShareLink
	err = pool.QueryRow(ctx, `
INSERT INTO transcripts.share_links (
  order_item_id, document_id, token, expires_at, max_downloads
) VALUES ($1, $2, $3, $4, $5)
RETURNING id, order_item_id, document_id, token, expires_at, max_downloads,
          download_count, opened_at, last_ip, created_at
`, itemID, documentID, token, expiresAt.UTC(), maxDownloads).Scan(
		&sl.ID, &sl.OrderItemID, &sl.DocumentID, &sl.Token, &sl.ExpiresAt, &sl.MaxDownloads,
		&sl.DownloadCount, &sl.OpenedAt, &sl.LastIP, &sl.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// GetShareLinkByToken loads a share link by opaque token.
func GetShareLinkByToken(ctx context.Context, pool *pgxpool.Pool, token string) (*ShareLink, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrShareLinkNotFound
	}
	var sl ShareLink
	err := pool.QueryRow(ctx, `
SELECT id, order_item_id, document_id, token, expires_at, max_downloads,
       download_count, opened_at, last_ip, created_at
FROM transcripts.share_links WHERE token = $1
`, token).Scan(
		&sl.ID, &sl.OrderItemID, &sl.DocumentID, &sl.Token, &sl.ExpiresAt, &sl.MaxDownloads,
		&sl.DownloadCount, &sl.OpenedAt, &sl.LastIP, &sl.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShareLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// RecordShareLinkOpen marks first open and stores requester IP.
func RecordShareLinkOpen(ctx context.Context, pool *pgxpool.Pool, linkID uuid.UUID, ip string) (*ShareLink, error) {
	var sl ShareLink
	err := pool.QueryRow(ctx, `
UPDATE transcripts.share_links
SET opened_at = COALESCE(opened_at, NOW()),
    last_ip = NULLIF(TRIM($2), '')
WHERE id = $1
RETURNING id, order_item_id, document_id, token, expires_at, max_downloads,
          download_count, opened_at, last_ip, created_at
`, linkID, ip).Scan(
		&sl.ID, &sl.OrderItemID, &sl.DocumentID, &sl.Token, &sl.ExpiresAt, &sl.MaxDownloads,
		&sl.DownloadCount, &sl.OpenedAt, &sl.LastIP, &sl.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShareLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// ConsumeShareLinkDownload increments download_count when under cap and not expired.
func ConsumeShareLinkDownload(ctx context.Context, pool *pgxpool.Pool, token string, now time.Time, ip string) (*ShareLink, error) {
	var sl ShareLink
	err := pool.QueryRow(ctx, `
UPDATE transcripts.share_links
SET download_count = download_count + 1,
    opened_at = COALESCE(opened_at, NOW()),
    last_ip = COALESCE(NULLIF(TRIM($3), ''), last_ip)
WHERE token = $1
  AND expires_at > $2
  AND download_count < max_downloads
RETURNING id, order_item_id, document_id, token, expires_at, max_downloads,
          download_count, opened_at, last_ip, created_at
`, token, now.UTC(), ip).Scan(
		&sl.ID, &sl.OrderItemID, &sl.DocumentID, &sl.Token, &sl.ExpiresAt, &sl.MaxDownloads,
		&sl.DownloadCount, &sl.OpenedAt, &sl.LastIP, &sl.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, gerr := GetShareLinkByToken(ctx, pool, token)
		if gerr != nil {
			return nil, gerr
		}
		if !existing.ExpiresAt.After(now.UTC()) {
			return nil, ErrShareLinkExpired
		}
		if existing.DownloadCount >= existing.MaxDownloads {
			return nil, ErrShareLinkExhausted
		}
		return nil, ErrShareLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// LatestShareLinkForItem returns the newest share link for an item, if any.
func LatestShareLinkForItem(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (*ShareLink, error) {
	var sl ShareLink
	err := pool.QueryRow(ctx, `
SELECT id, order_item_id, document_id, token, expires_at, max_downloads,
       download_count, opened_at, last_ip, created_at
FROM transcripts.share_links
WHERE order_item_id = $1
ORDER BY created_at DESC
LIMIT 1
`, itemID).Scan(
		&sl.ID, &sl.OrderItemID, &sl.DocumentID, &sl.Token, &sl.ExpiresAt, &sl.MaxDownloads,
		&sl.DownloadCount, &sl.OpenedAt, &sl.LastIP, &sl.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShareLinkNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// InsertPostalJob queues a postal fulfillment job.
func InsertPostalJob(
	ctx context.Context,
	pool *pgxpool.Pool,
	itemID, documentID uuid.UUID,
	address json.RawMessage,
) (*PostalJob, error) {
	if len(address) == 0 {
		address = json.RawMessage(`{}`)
	}
	var j PostalJob
	err := pool.QueryRow(ctx, `
INSERT INTO transcripts.postal_jobs (order_item_id, document_id, address, status)
VALUES ($1, $2, $3::jsonb, 'queued')
RETURNING id, order_item_id, document_id, address, status, vendor_ref, detail, created_at, updated_at
`, itemID, documentID, address).Scan(
		&j.ID, &j.OrderItemID, &j.DocumentID, &j.Address, &j.Status, &j.VendorRef, &j.Detail,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// RecordOpenedReceipt appends an opened receipt attempt row (download page open).
func RecordOpenedReceipt(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, adapter DeliveryMethod, detail string) error {
	n, err := NextAttemptNo(ctx, pool, itemID)
	if err != nil {
		return err
	}
	a, err := InsertDeliveryAttempt(ctx, pool, itemID, adapter, n)
	if err != nil {
		return err
	}
	d := detail
	_, err = UpdateDeliveryAttemptStatus(ctx, pool, a.ID, AttemptOpened, nil, &d)
	return err
}
