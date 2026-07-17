package transcripts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/transcriptfees"
	"github.com/lextures/lextures/server/internal/models/transcriptorder"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// Order payment status aliases (stored on transcripts.orders.payment_status).
const (
	OrderPaymentUnpaid            = transcriptfees.PaymentUnpaid
	OrderPaymentPending           = transcriptfees.PaymentPending
	OrderPaymentPaid              = transcriptfees.PaymentPaid
	OrderPaymentWaived            = transcriptfees.PaymentWaived
	OrderPaymentRefunded          = transcriptfees.PaymentRefunded
	OrderPaymentPartiallyRefunded = transcriptfees.PaymentPartiallyRefunded
	OrderPaymentFree              = transcriptfees.PaymentFree
)

var (
	ErrFeeScheduleNotFound = errors.New("fee schedule not found")
	ErrWaiverCodeNotFound  = errors.New("waiver code not found")
	ErrWaiverCodeInvalid   = errors.New("waiver code invalid or expired")
	ErrWaiverCodeExhausted = errors.New("waiver code has no uses remaining")
	ErrPaymentNotRequired  = errors.New("payment not required")
	ErrPaymentAlreadyDone  = errors.New("payment already satisfied")
	ErrRefundNotAllowed    = errors.New("refund not allowed for order state")
	ErrCheckoutNotReady    = errors.New("order not ready for checkout")
)

// FeeSchedule is the org transcript fee schedule.
type FeeSchedule struct {
	OrgID            uuid.UUID
	Currency         string
	BaseFee          int
	RushFee          int
	PerRecipientFee  int
	MethodSurcharges map[string]int
	FreeAllotment    int
	AllotmentPeriod  string
	UpdatedAt        time.Time
}

// WaiverCode is a reusable/limited waiver code.
type WaiverCode struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	Code       string
	Kind       string
	Value      *int
	MaxUses    *int
	UsedCount  int
	ExpiresAt  *time.Time
	CreatedBy  *uuid.UUID
	CreatedAt  time.Time
}

// UpsertFeeScheduleInput patches the org fee schedule.
type UpsertFeeScheduleInput struct {
	OrgID            uuid.UUID
	Currency         string
	BaseFee          int
	RushFee          int
	PerRecipientFee  int
	MethodSurcharges map[string]int
	FreeAllotment    int
	AllotmentPeriod  string
}

// CreateWaiverCodeInput creates a new waiver code.
type CreateWaiverCodeInput struct {
	OrgID     uuid.UUID
	Code      string
	Kind      string
	Value     *int
	MaxUses   *int
	ExpiresAt *time.Time
	CreatedBy *uuid.UUID
}

// QuoteOptions controls quote evaluation for an order.
type QuoteOptions struct {
	WaiverCode string
	// ApplyFreeAllotment defaults true when free allotment remains.
	SkipFreeAllotment bool
}

func defaultFeeSchedule(orgID uuid.UUID) *FeeSchedule {
	return &FeeSchedule{
		OrgID:            orgID,
		Currency:         "usd",
		MethodSurcharges: map[string]int{},
		AllotmentPeriod:  "lifetime",
	}
}

// GetFeeSchedule returns the org schedule or zeros when missing.
func GetFeeSchedule(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*FeeSchedule, error) {
	var s FeeSchedule
	var raw []byte
	err := pool.QueryRow(ctx, `
SELECT org_id, currency, base_fee, rush_fee, per_recipient_fee, method_surcharges,
       free_allotment, allotment_period, updated_at
FROM transcripts.fee_schedule
WHERE org_id = $1
`, orgID).Scan(
		&s.OrgID, &s.Currency, &s.BaseFee, &s.RushFee, &s.PerRecipientFee, &raw,
		&s.FreeAllotment, &s.AllotmentPeriod, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultFeeSchedule(orgID), nil
	}
	if err != nil {
		return nil, err
	}
	s.MethodSurcharges = map[string]int{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s.MethodSurcharges)
	}
	if s.MethodSurcharges == nil {
		s.MethodSurcharges = map[string]int{}
	}
	return &s, nil
}

// UpsertFeeSchedule saves the org fee schedule.
func UpsertFeeSchedule(ctx context.Context, pool *pgxpool.Pool, in UpsertFeeScheduleInput) (*FeeSchedule, error) {
	cur := strings.ToLower(strings.TrimSpace(in.Currency))
	if cur == "" {
		cur = "usd"
	}
	period := strings.ToLower(strings.TrimSpace(in.AllotmentPeriod))
	if period == "" {
		period = "lifetime"
	}
	switch period {
	case "lifetime", "year", "term":
	default:
		return nil, fmt.Errorf("invalid allotment_period %q", period)
	}
	if in.BaseFee < 0 || in.RushFee < 0 || in.PerRecipientFee < 0 || in.FreeAllotment < 0 {
		return nil, errors.New("fee amounts must be non-negative")
	}
	surcharges := in.MethodSurcharges
	if surcharges == nil {
		surcharges = map[string]int{}
	}
	raw, err := json.Marshal(surcharges)
	if err != nil {
		return nil, err
	}
	var s FeeSchedule
	var outRaw []byte
	err = pool.QueryRow(ctx, `
INSERT INTO transcripts.fee_schedule (
    org_id, currency, base_fee, rush_fee, per_recipient_fee, method_surcharges,
    free_allotment, allotment_period, updated_at
) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, NOW())
ON CONFLICT (org_id) DO UPDATE SET
    currency = EXCLUDED.currency,
    base_fee = EXCLUDED.base_fee,
    rush_fee = EXCLUDED.rush_fee,
    per_recipient_fee = EXCLUDED.per_recipient_fee,
    method_surcharges = EXCLUDED.method_surcharges,
    free_allotment = EXCLUDED.free_allotment,
    allotment_period = EXCLUDED.allotment_period,
    updated_at = NOW()
RETURNING org_id, currency, base_fee, rush_fee, per_recipient_fee, method_surcharges,
          free_allotment, allotment_period, updated_at
`, in.OrgID, cur, in.BaseFee, in.RushFee, in.PerRecipientFee, string(raw),
		in.FreeAllotment, period).Scan(
		&s.OrgID, &s.Currency, &s.BaseFee, &s.RushFee, &s.PerRecipientFee, &outRaw,
		&s.FreeAllotment, &s.AllotmentPeriod, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.MethodSurcharges = map[string]int{}
	_ = json.Unmarshal(outRaw, &s.MethodSurcharges)
	return &s, nil
}

// ListWaiverCodes returns waiver codes for an org.
func ListWaiverCodes(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]WaiverCode, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, code, kind, value, max_uses, used_count, expires_at, created_by, created_at
FROM transcripts.waiver_codes
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT 200
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WaiverCode
	for rows.Next() {
		var c WaiverCode
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.Code, &c.Kind, &c.Value, &c.MaxUses, &c.UsedCount,
			&c.ExpiresAt, &c.CreatedBy, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateWaiverCode inserts a new waiver code.
func CreateWaiverCode(ctx context.Context, pool *pgxpool.Pool, in CreateWaiverCodeInput) (*WaiverCode, error) {
	code := strings.ToUpper(strings.TrimSpace(in.Code))
	if code == "" {
		return nil, errors.New("code is required")
	}
	kind := strings.ToLower(strings.TrimSpace(in.Kind))
	switch kind {
	case "full", "percent", "amount":
	default:
		return nil, fmt.Errorf("invalid waiver kind %q", in.Kind)
	}
	var c WaiverCode
	err := pool.QueryRow(ctx, `
INSERT INTO transcripts.waiver_codes (org_id, code, kind, value, max_uses, expires_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, code, kind, value, max_uses, used_count, expires_at, created_by, created_at
`, in.OrgID, code, kind, in.Value, in.MaxUses, in.ExpiresAt, in.CreatedBy).Scan(
		&c.ID, &c.OrgID, &c.Code, &c.Kind, &c.Value, &c.MaxUses, &c.UsedCount,
		&c.ExpiresAt, &c.CreatedBy, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// FindWaiverCode looks up a code for an org (case-insensitive).
func FindWaiverCode(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, code string) (*WaiverCode, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, ErrWaiverCodeNotFound
	}
	var c WaiverCode
	err := pool.QueryRow(ctx, `
SELECT id, org_id, code, kind, value, max_uses, used_count, expires_at, created_by, created_at
FROM transcripts.waiver_codes
WHERE org_id = $1 AND code = $2
`, orgID, code).Scan(
		&c.ID, &c.OrgID, &c.Code, &c.Kind, &c.Value, &c.MaxUses, &c.UsedCount,
		&c.ExpiresAt, &c.CreatedBy, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWaiverCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *WaiverCode) validateUsable() error {
	if c == nil {
		return ErrWaiverCodeNotFound
	}
	if c.ExpiresAt != nil && !c.ExpiresAt.After(time.Now().UTC()) {
		return ErrWaiverCodeInvalid
	}
	if c.MaxUses != nil && c.UsedCount >= *c.MaxUses {
		return ErrWaiverCodeExhausted
	}
	return nil
}

func (c *WaiverCode) toWaiverInput() *transcriptfees.WaiverInput {
	if c == nil {
		return nil
	}
	w := &transcriptfees.WaiverInput{Kind: transcriptfees.WaiverKind(c.Kind)}
	if c.Value != nil {
		w.Value = *c.Value
	}
	return w
}

// CountFreeAllotmentUsed counts orders that consumed free allotment in the period.
func CountFreeAllotmentUsed(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	orgID *uuid.UUID,
	period string,
) (int, error) {
	q := `
SELECT COUNT(*)::int
FROM transcripts.orders
WHERE user_id = $1
  AND free_allotment_applied = TRUE
  AND status NOT IN ('canceled', 'rejected', 'failed')
`
	args := []any{userID}
	if orgID != nil {
		q += ` AND org_id = $2`
		args = append(args, *orgID)
		switch strings.ToLower(period) {
		case "year":
			q += ` AND created_at >= date_trunc('year', NOW())`
		case "term":
			// Approximate academic term as rolling 4 months.
			q += ` AND created_at >= NOW() - INTERVAL '4 months'`
		}
	} else {
		switch strings.ToLower(period) {
		case "year":
			q += ` AND created_at >= date_trunc('year', NOW())`
		case "term":
			q += ` AND created_at >= NOW() - INTERVAL '4 months'`
		}
	}
	var n int
	if err := pool.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// QuoteOrder computes an itemized quote for an order (optionally applying a waiver code).
func QuoteOrder(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *Config,
	o *Order,
	opts QuoteOptions,
) (*transcriptfees.Quote, *WaiverCode, error) {
	if o == nil {
		return nil, nil, ErrOrderNotFound
	}
	if cfg == nil || !cfg.FeesEnabled {
		q := transcriptfees.Quote{
			Currency:            "usd",
			Lines:               []transcriptfees.QuoteLine{},
			Total:               0,
			RequiresPayment:     false,
			PaymentStatusIfZero: transcriptfees.PaymentFree,
		}
		return &q, nil, nil
	}
	orgID := uuid.Nil
	if o.OrgID != nil {
		orgID = *o.OrgID
	}
	sched, err := GetFeeSchedule(ctx, pool, orgID)
	if err != nil {
		return nil, nil, err
	}
	items := make([]transcriptfees.LineItem, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, transcriptfees.LineItem{
			DeliveryMethod: string(it.DeliveryMethod),
			Urgency:        string(it.Urgency),
		})
	}
	remain := 0
	if sched.FreeAllotment > 0 {
		used, err := CountFreeAllotmentUsed(ctx, pool, o.UserID, o.OrgID, sched.AllotmentPeriod)
		if err != nil {
			return nil, nil, err
		}
		remain = sched.FreeAllotment - used
		if remain < 0 {
			remain = 0
		}
	}
	var waiverCode *WaiverCode
	var waiverIn *transcriptfees.WaiverInput
	if code := strings.TrimSpace(opts.WaiverCode); code != "" {
		if orgID == uuid.Nil {
			return nil, nil, ErrWaiverCodeNotFound
		}
		wc, err := FindWaiverCode(ctx, pool, orgID, code)
		if err != nil {
			return nil, nil, err
		}
		if err := wc.validateUsable(); err != nil {
			return nil, nil, err
		}
		waiverCode = wc
		waiverIn = wc.toWaiverInput()
	} else if o.WaiverID != nil {
		wc, err := getWaiverCodeByID(ctx, pool, *o.WaiverID)
		if err == nil {
			waiverCode = wc
			waiverIn = wc.toWaiverInput()
		}
	}
	q := transcriptfees.ComputeQuote(transcriptfees.QuoteInput{
		Schedule: transcriptfees.Schedule{
			Currency:         sched.Currency,
			BaseFee:          sched.BaseFee,
			RushFee:          sched.RushFee,
			PerRecipientFee:  sched.PerRecipientFee,
			MethodSurcharges: sched.MethodSurcharges,
			FreeAllotment:    sched.FreeAllotment,
			AllotmentPeriod:  transcriptfees.AllotmentPeriod(sched.AllotmentPeriod),
		},
		Items:               items,
		Waiver:              waiverIn,
		FreeAllotmentRemain: remain,
		ApplyFreeAllotment:  !opts.SkipFreeAllotment && remain > 0 && waiverIn == nil,
	})
	return &q, waiverCode, nil
}

func getWaiverCodeByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*WaiverCode, error) {
	var c WaiverCode
	err := pool.QueryRow(ctx, `
SELECT id, org_id, code, kind, value, max_uses, used_count, expires_at, created_by, created_at
FROM transcripts.waiver_codes
WHERE id = $1
`, id).Scan(
		&c.ID, &c.OrgID, &c.Code, &c.Kind, &c.Value, &c.MaxUses, &c.UsedCount,
		&c.ExpiresAt, &c.CreatedBy, &c.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWaiverCodeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// PaymentSatisfiedForOrder evaluates the T05 payment gate.
func PaymentSatisfiedForOrder(ctx context.Context, pool *pgxpool.Pool, cfg *Config, o *Order) (bool, error) {
	if cfg == nil || !cfg.FeesEnabled {
		return true, nil
	}
	if o == nil {
		return false, ErrOrderNotFound
	}
	st, err := transcriptfees.ParsePaymentStatus(string(o.PaymentStatus))
	if err != nil {
		// Legacy rows without status treated as unpaid when fees on.
		return false, nil
	}
	if st.SatisfiesPaymentGate() {
		return true, nil
	}
	// Zero-total schedules: auto-treat as free when still unpaid (no checkout needed).
	if st == transcriptfees.PaymentUnpaid {
		q, _, qerr := QuoteOrder(ctx, pool, cfg, o, QuoteOptions{})
		if qerr != nil {
			return false, qerr
		}
		if q != nil && !q.RequiresPayment {
			return true, nil
		}
	}
	return false, nil
}

// PersistOrderQuote writes total/currency and optionally applies zero-total payment state.
func PersistOrderQuote(
	ctx context.Context,
	pool *pgxpool.Pool,
	orderID uuid.UUID,
	q *transcriptfees.Quote,
	waiverID *uuid.UUID,
	markZeroPaid bool,
) error {
	if q == nil {
		return nil
	}
	paymentStatus := ""
	freeApplied := false
	if markZeroPaid && !q.RequiresPayment {
		if q.FreeAllotmentApplied {
			paymentStatus = string(transcriptfees.PaymentFree)
			freeApplied = true
		} else if q.PaymentStatusIfZero != "" {
			paymentStatus = string(q.PaymentStatusIfZero)
		} else {
			paymentStatus = string(transcriptfees.PaymentFree)
		}
	}
	if paymentStatus != "" {
		_, err := pool.Exec(ctx, `
UPDATE transcripts.orders
SET total_amount = $2,
    currency = $3,
    waiver_id = COALESCE($4, waiver_id),
    payment_status = $5,
    free_allotment_applied = $6
WHERE id = $1
`, orderID, q.Total, q.Currency, waiverID, paymentStatus, freeApplied)
		return err
	}
	_, err := pool.Exec(ctx, `
UPDATE transcripts.orders
SET total_amount = $2,
    currency = $3,
    waiver_id = COALESCE($4, waiver_id)
WHERE id = $1
`, orderID, q.Total, q.Currency, waiverID)
	return err
}

// ApplyWaiverCodeToOrder validates a code, persists quote, and may mark waived/free.
func ApplyWaiverCodeToOrder(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *Config,
	o *Order,
	code string,
	actorID *uuid.UUID,
) (*transcriptfees.Quote, error) {
	q, wc, err := QuoteOrder(ctx, pool, cfg, o, QuoteOptions{WaiverCode: code, SkipFreeAllotment: true})
	if err != nil {
		return nil, err
	}
	if wc == nil {
		return nil, ErrWaiverCodeNotFound
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Increment use count with concurrency guard.
	tag, err := tx.Exec(ctx, `
UPDATE transcripts.waiver_codes
SET used_count = used_count + 1
WHERE id = $1
  AND (max_uses IS NULL OR used_count < max_uses)
  AND (expires_at IS NULL OR expires_at > NOW())
`, wc.ID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrWaiverCodeExhausted
	}

	status := string(transcriptfees.PaymentUnpaid)
	if !q.RequiresPayment {
		status = string(transcriptfees.PaymentWaived)
	}
	if _, err := tx.Exec(ctx, `
UPDATE transcripts.orders
SET total_amount = $2,
    currency = $3,
    waiver_id = $4,
    payment_status = CASE
        WHEN payment_status IN ('paid', 'refunded', 'partially_refunded') THEN payment_status
        ELSE $5
    END
WHERE id = $1
`, o.ID, q.Total, q.Currency, wc.ID, status); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.waiver_applications (
    order_id, org_id, waiver_code_id, kind, value, amount_waived, reason, applied_by
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, o.ID, o.OrgID, wc.ID, wc.Kind, wc.Value, q.WaiverAmount, "waiver code applied", actorID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	telemetry.RecordBusinessEvent("transcript_waiver_applied")
	return q, nil
}

// AdminWaiveOrder grants a full admin waiver (zeros total).
func AdminWaiveOrder(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *Config,
	orderID uuid.UUID,
	actorID uuid.UUID,
	reason string,
) (*Order, error) {
	o, err := GetOrderByID(ctx, pool, orderID)
	if err != nil {
		return nil, err
	}
	q, _, err := QuoteOrder(ctx, pool, cfg, o, QuoteOptions{SkipFreeAllotment: true})
	if err != nil {
		return nil, err
	}
	waived := q.Subtotal
	if waived == 0 && o.TotalAmount != nil {
		waived = *o.TotalAmount
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "admin waiver"
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
UPDATE transcripts.orders
SET payment_status = 'waived',
    total_amount = 0,
    currency = COALESCE(currency, 'usd')
WHERE id = $1
`, orderID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.waiver_applications (
    order_id, org_id, kind, amount_waived, reason, applied_by
) VALUES ($1, $2, 'admin', $3, $4, $5)
`, orderID, o.OrgID, waived, reason, actorID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	telemetry.RecordBusinessEvent("transcript_waiver_applied")
	// Advance past pending_payment if stuck there.
	return AdvanceAfterPayment(ctx, pool, cfg, orderID, &actorID)
}

// MarkOrderPaymentPending stores checkout session/PI reference.
func MarkOrderPaymentPending(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID, paymentRef string, total int, currency string) error {
	_, err := pool.Exec(ctx, `
UPDATE transcripts.orders
SET payment_status = 'pending',
    payment_ref = $2,
    total_amount = $3,
    currency = $4
WHERE id = $1
`, orderID, paymentRef, total, currency)
	return err
}

// RecordPaymentEvent inserts an idempotent Stripe event row. Returns false if duplicate.
func RecordPaymentEvent(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID, stripeEventID, eventType string, payload []byte) (bool, error) {
	tag, err := pool.Exec(ctx, `
INSERT INTO transcripts.payment_events (order_id, stripe_event_id, event_type, payload)
VALUES ($1, $2, $3, COALESCE($4::jsonb, '{}'::jsonb))
ON CONFLICT (stripe_event_id) DO NOTHING
`, orderID, stripeEventID, eventType, nullableJSONBytes(payload))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func nullableJSONBytes(b []byte) *string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	return &s
}

// MarkOrderPaidFromStripe sets paid state and payment_ref (PaymentIntent preferred).
func MarkOrderPaidFromStripe(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *Config,
	orderID uuid.UUID,
	paymentRef string,
	amount int,
	currency string,
	stripeEventID string,
	rawPayload []byte,
) (*Order, error) {
	created, err := RecordPaymentEvent(ctx, pool, orderID, stripeEventID, "checkout.session.completed", rawPayload)
	if err != nil {
		return nil, err
	}
	if !created {
		// Idempotent retry — still return current order.
		return GetOrderByID(ctx, pool, orderID)
	}
	cur := strings.ToLower(strings.TrimSpace(currency))
	if cur == "" {
		cur = "usd"
	}
	if _, err := pool.Exec(ctx, `
UPDATE transcripts.orders
SET payment_status = 'paid',
    payment_ref = COALESCE(NULLIF(TRIM($2), ''), payment_ref),
    total_amount = $3,
    currency = $4
WHERE id = $1
`, orderID, paymentRef, amount, cur); err != nil {
		return nil, err
	}
	telemetry.RecordBusinessEvent("transcript_payment_succeeded")
	return AdvanceAfterPayment(ctx, pool, cfg, orderID, nil)
}

// MarkOrderRefundedFromStripe updates refund amounts idempotently.
func MarkOrderRefundedFromStripe(
	ctx context.Context,
	pool *pgxpool.Pool,
	orderID uuid.UUID,
	amountRefunded int,
	stripeEventID string,
	rawPayload []byte,
) (*Order, error) {
	created, err := RecordPaymentEvent(ctx, pool, orderID, stripeEventID, "charge.refunded", rawPayload)
	if err != nil {
		return nil, err
	}
	if !created {
		return GetOrderByID(ctx, pool, orderID)
	}
	o, err := GetOrderByID(ctx, pool, orderID)
	if err != nil {
		return nil, err
	}
	total := 0
	if o.TotalAmount != nil {
		total = *o.TotalAmount
	}
	status := string(transcriptfees.PaymentPartiallyRefunded)
	if amountRefunded >= total && total > 0 {
		status = string(transcriptfees.PaymentRefunded)
	}
	if _, err := pool.Exec(ctx, `
UPDATE transcripts.orders
SET amount_refunded = GREATEST(amount_refunded, $2),
    payment_status = $3
WHERE id = $1
`, orderID, amountRefunded, status); err != nil {
		return nil, err
	}
	telemetry.RecordBusinessEvent("transcript_payment_refunded")
	return GetOrderByID(ctx, pool, orderID)
}

// ApplyAdminRefund records a local refund state after Stripe refund succeeds.
func ApplyAdminRefund(
	ctx context.Context,
	pool *pgxpool.Pool,
	orderID uuid.UUID,
	amount int,
) (*Order, error) {
	o, err := GetOrderByID(ctx, pool, orderID)
	if err != nil {
		return nil, err
	}
	if o.PaymentStatus != OrderPaymentPaid && o.PaymentStatus != OrderPaymentPartiallyRefunded {
		return nil, ErrRefundNotAllowed
	}
	total := 0
	if o.TotalAmount != nil {
		total = *o.TotalAmount
	}
	newRefunded := o.AmountRefunded + amount
	if amount <= 0 {
		newRefunded = total
	}
	if newRefunded > total && total > 0 {
		newRefunded = total
	}
	status := string(transcriptfees.PaymentPartiallyRefunded)
	if total == 0 || newRefunded >= total {
		status = string(transcriptfees.PaymentRefunded)
	}
	if _, err := pool.Exec(ctx, `
UPDATE transcripts.orders
SET amount_refunded = $2,
    payment_status = $3
WHERE id = $1
`, orderID, newRefunded, status); err != nil {
		return nil, err
	}
	telemetry.RecordBusinessEvent("transcript_payment_refunded")
	return GetOrderByID(ctx, pool, orderID)
}

// FindOrderIDByPaymentRef resolves an order from Stripe PI or session id.
func FindOrderIDByPaymentRef(ctx context.Context, pool *pgxpool.Pool, ref string) (uuid.UUID, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return uuid.Nil, ErrOrderNotFound
	}
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id FROM transcripts.orders WHERE payment_ref = $1 LIMIT 1
`, ref).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrOrderNotFound
	}
	return id, err
}

// AdvanceAfterPayment moves pending_payment orders forward when payment is satisfied.
func AdvanceAfterPayment(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *Config,
	orderID uuid.UUID,
	actorID *uuid.UUID,
) (*Order, error) {
	o, err := GetOrderByID(ctx, pool, orderID)
	if err != nil {
		return nil, err
	}
	if o.Status != OrderPendingPayment && o.Status != OrderPendingConsent {
		return o, nil
	}
	gates, err := GateContextForOrder(ctx, pool, o, cfg != nil && cfg.AutoApprovalEnabled)
	if err != nil {
		return nil, err
	}
	target := transcriptorder.ResolveSubmitTarget(gates)
	if target == transcriptorder.OrderPendingPayment || target == transcriptorder.OrderPendingConsent {
		return o, nil
	}
	reason := "payment satisfied"
	switch o.PaymentStatus {
	case OrderPaymentWaived:
		reason = "fee waived"
	case OrderPaymentFree:
		reason = "no payment due"
	}
	if _, err := transitionOrderTx(ctx, pool, transitionParams{
		OrderID:        orderID,
		ActorID:        actorID,
		From:           o.Status,
		To:             OrderStatus(target),
		Reason:         &reason,
		MarkItemsReady: target == transcriptorder.OrderProcessing,
	}); err != nil {
		// Illegal transition (e.g. consent still pending) — return current.
		if errors.Is(err, ErrIllegalOrderTransition) {
			return GetOrderByID(ctx, pool, orderID)
		}
		return nil, err
	}
	return GetOrderByID(ctx, pool, orderID)
}
