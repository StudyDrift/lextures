// Coupon redemption ledger: reserve / redeem / release (plan MKTC.1).
package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/coupons"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// Redemption statuses.
const (
	RedemptionReserved = "reserved"
	RedemptionRedeemed = "redeemed"
	RedemptionReleased = "released"
)

// CouponReservationTTL is how long a checkout reservation holds a seat (FR-7).
// Open Q2: default 30 minutes.
const CouponReservationTTL = 30 * time.Minute

// Redemption is a row in billing.coupon_redemptions.
type Redemption struct {
	ID                uuid.UUID
	CouponID          uuid.UUID
	CourseID          uuid.UUID
	UserID            uuid.UUID
	EntitlementID     *uuid.UUID
	Status            string
	CheckoutSessionID *string
	ProviderEventID   *string
	ListPriceCents    int
	DiscountCents     int
	ChargedCents      int
	Currency          string
	ReservedAt        time.Time
	ExpiresAt         *time.Time
	RedeemedAt        *time.Time
	ReleasedAt        *time.Time
}

// ReserveInput is the payload for ReserveCoupon.
type ReserveInput struct {
	CouponID          uuid.UUID
	UserID            uuid.UUID
	CoursePriceCents  int
	CourseCurrency    string
	AlreadyOwned      bool
	CheckoutSessionID *string
	Now               time.Time // zero → time.Now().UTC()
}

// RedeemInput promotes a reservation to redeemed (idempotent on ProviderEventID).
type RedeemInput struct {
	RedemptionID    uuid.UUID
	ProviderEventID string
	EntitlementID   *uuid.UUID
	// Or look up by checkout session when RedemptionID is nil/zero:
	CheckoutSessionID string
}

const redemptionSelectCols = `
id, coupon_id, course_id, user_id, entitlement_id, status,
checkout_session_id, provider_event_id,
list_price_cents, discount_cents, charged_cents, currency,
reserved_at, expires_at, redeemed_at, released_at
`

func scanRedemption(row pgx.Row) (*Redemption, error) {
	var r Redemption
	err := row.Scan(
		&r.ID, &r.CouponID, &r.CourseID, &r.UserID, &r.EntitlementID, &r.Status,
		&r.CheckoutSessionID, &r.ProviderEventID,
		&r.ListPriceCents, &r.DiscountCents, &r.ChargedCents, &r.Currency,
		&r.ReservedAt, &r.ExpiresAt, &r.RedeemedAt, &r.ReleasedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ReserveCoupon locks the coupon row, re-checks eligibility against live counts,
// and inserts a reserved redemption with expires_at = now + TTL (FR-7).
// Concurrency correctness comes from SELECT … FOR UPDATE on the coupon row.
func ReserveCoupon(ctx context.Context, pool *pgxpool.Pool, in ReserveInput) (*Redemption, coupons.Reason, error) {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock coupon row.
	c, err := scanCoupon(tx.QueryRow(ctx, `
SELECT `+couponSelectCols+`
FROM billing.course_coupons
WHERE id = $1
FOR UPDATE
`, in.CouponID))
	if errors.Is(err, pgx.ErrNoRows) {
		telemetry.RecordCouponReserve("not_found")
		return nil, coupons.ReasonNotFound, nil
	}
	if err != nil {
		return nil, "", err
	}

	// Live seat counts under the same lock.
	var consumed, userSeats int
	if err := tx.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM billing.coupon_redemptions
WHERE coupon_id = $1
  AND (
    status = 'redeemed'
    OR (status = 'reserved' AND (expires_at IS NULL OR expires_at > $2))
  )
`, c.ID, now).Scan(&consumed); err != nil {
		return nil, "", err
	}
	if err := tx.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM billing.coupon_redemptions
WHERE coupon_id = $1 AND user_id = $2
  AND (
    status = 'redeemed'
    OR (status = 'reserved' AND (expires_at IS NULL OR expires_at > $3))
  )
`, c.ID, in.UserID, now).Scan(&userSeats); err != nil {
		return nil, "", err
	}

	domain := c.ToDomain()
	reason, quote := coupons.Evaluate(&domain, coupons.EvalContext{
		Now:            now,
		CoursePrice:    in.CoursePriceCents,
		CourseCurrency: in.CourseCurrency,
		ConsumedSeats:  consumed,
		UserSeats:      userSeats,
		AlreadyOwned:   in.AlreadyOwned,
	})
	if reason != coupons.ReasonOK {
		telemetry.RecordCouponReserve(string(reason))
		return nil, reason, nil
	}

	expires := now.Add(CouponReservationTTL)
	var session any
	if in.CheckoutSessionID != nil && *in.CheckoutSessionID != "" {
		session = *in.CheckoutSessionID
	}

	r, err := scanRedemption(tx.QueryRow(ctx, `
INSERT INTO billing.coupon_redemptions (
    coupon_id, course_id, user_id, status, checkout_session_id,
    list_price_cents, discount_cents, charged_cents, currency,
    reserved_at, expires_at
) VALUES ($1,$2,$3,'reserved',$4,$5,$6,$7,$8,$9,$10)
RETURNING `+redemptionSelectCols,
		c.ID, c.CourseID, in.UserID, session,
		quote.ListCents, quote.DiscountCents, quote.ChargedCents, quote.Currency,
		now, expires,
	))
	if err != nil {
		// Unique on checkout_session_id: return existing reservation if same session.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && session != nil {
			existing, getErr := scanRedemption(tx.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions WHERE checkout_session_id = $1
`, session))
			if getErr == nil {
				if err := tx.Commit(ctx); err != nil {
					return nil, "", err
				}
				telemetry.RecordCouponReserve("ok")
				return existing, coupons.ReasonOK, nil
			}
		}
		return nil, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", err
	}
	telemetry.RecordCouponReserve("ok")
	return r, coupons.ReasonOK, nil
}

// RedeemCoupon promotes a reservation to redeemed. Idempotent on provider_event_id
// (FR-8, AC-7). Increments course_coupons.redeemed_count once per new redeem.
// Returns (redemption, created, error) where created is false on idempotent replay.
func RedeemCoupon(ctx context.Context, pool *pgxpool.Pool, in RedeemInput) (*Redemption, bool, error) {
	if in.ProviderEventID == "" {
		return nil, false, errors.New("provider_event_id required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Idempotent: existing row for this event.
	existing, err := scanRedemption(tx.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions WHERE provider_event_id = $1
`, in.ProviderEventID))
	if err == nil && existing != nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, false, err
		}
		telemetry.RecordCouponRedeem("idempotent")
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	// Locate the reservation.
	var row pgx.Row
	switch {
	case in.RedemptionID != uuid.Nil:
		row = tx.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions WHERE id = $1 FOR UPDATE
`, in.RedemptionID)
	case in.CheckoutSessionID != "":
		row = tx.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions WHERE checkout_session_id = $1 FOR UPDATE
`, in.CheckoutSessionID)
	default:
		return nil, false, errors.New("redemption_id or checkout_session_id required")
	}

	r, err := scanRedemption(row)
	if errors.Is(err, pgx.ErrNoRows) {
		telemetry.RecordCouponRedeem("not_found")
		return nil, false, pgx.ErrNoRows
	}
	if err != nil {
		return nil, false, err
	}

	if r.Status == RedemptionRedeemed {
		// Already redeemed (possibly without this event id) — attach event if missing.
		if r.ProviderEventID == nil || *r.ProviderEventID == "" {
			_, _ = tx.Exec(ctx, `
UPDATE billing.coupon_redemptions SET provider_event_id = $2 WHERE id = $1
`, r.ID, in.ProviderEventID)
			r.ProviderEventID = &in.ProviderEventID
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, false, err
		}
		telemetry.RecordCouponRedeem("idempotent")
		return r, false, nil
	}
	// reserved → redeemed, or released → redeemed (payment completed after TTL sweep — FR-10).
	if r.Status != RedemptionReserved && r.Status != RedemptionReleased {
		telemetry.RecordCouponRedeem("invalid_status")
		return nil, false, fmt.Errorf("cannot redeem redemption in status %q", r.Status)
	}
	wasReleased := r.Status == RedemptionReleased

	// Lock coupon for redeemed_count update.
	if _, err := tx.Exec(ctx, `
SELECT id FROM billing.course_coupons WHERE id = $1 FOR UPDATE
`, r.CouponID); err != nil {
		return nil, false, err
	}

	now := time.Now().UTC()
	updated, err := scanRedemption(tx.QueryRow(ctx, `
UPDATE billing.coupon_redemptions SET
    status = 'redeemed',
    provider_event_id = $2,
    entitlement_id = COALESCE($3, entitlement_id),
    redeemed_at = $4,
    expires_at = NULL,
    released_at = NULL
WHERE id = $1
RETURNING `+redemptionSelectCols,
		r.ID, in.ProviderEventID, in.EntitlementID, now,
	))
	if err != nil {
		return nil, false, err
	}

	// Increment redeemed_count for reserved→redeemed and for released→redeemed revival.
	if r.Status == RedemptionReserved || wasReleased {
		if _, err := tx.Exec(ctx, `
UPDATE billing.course_coupons
SET redeemed_count = redeemed_count + 1, updated_at = NOW()
WHERE id = $1
`, r.CouponID); err != nil {
			return nil, false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	telemetry.RecordCouponRedeem("ok")
	return updated, true, nil
}

// ReleaseCouponReservation moves a reserved (or, for refunds, redeemed) row to
// released and keeps redeemed_count in step when releasing a redeemed seat (FR-8).
func ReleaseCouponReservation(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, reason string) error {
	if reason == "" {
		reason = "manual"
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	r, err := scanRedemption(tx.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions WHERE id = $1 FOR UPDATE
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return pgx.ErrNoRows
	}
	if err != nil {
		return err
	}
	if r.Status == RedemptionReleased {
		telemetry.RecordCouponRelease(reason)
		return tx.Commit(ctx)
	}

	wasRedeemed := r.Status == RedemptionRedeemed
	if r.Status != RedemptionReserved && !wasRedeemed {
		return fmt.Errorf("cannot release redemption in status %q", r.Status)
	}

	if _, err := tx.Exec(ctx, `
SELECT id FROM billing.course_coupons WHERE id = $1 FOR UPDATE
`, r.CouponID); err != nil {
		return err
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
UPDATE billing.coupon_redemptions SET
    status = 'released',
    released_at = $2,
    expires_at = NULL
WHERE id = $1
`, id, now); err != nil {
		return err
	}

	if wasRedeemed {
		if _, err := tx.Exec(ctx, `
UPDATE billing.course_coupons
SET redeemed_count = GREATEST(0, redeemed_count - 1), updated_at = NOW()
WHERE id = $1
`, r.CouponID); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	telemetry.RecordCouponRelease(reason)
	return nil
}

// ReleaseExpiredCouponReservations marks expired reserved rows as released (FR-10).
// Correctness does not depend on this having run — seat counts exclude expired rows.
func ReleaseExpiredCouponReservations(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	tag, err := pool.Exec(ctx, `
UPDATE billing.coupon_redemptions
SET status = 'released', released_at = $1
WHERE status = 'reserved'
  AND expires_at IS NOT NULL
  AND expires_at <= $1
`, now)
	if err != nil {
		return 0, err
	}
	n := int(tag.RowsAffected())
	if n > 0 {
		telemetry.RecordCouponReservationExpired(n)
	}
	return n, nil
}
