// Immediate coupon redemption (free grants / webhook fallback) (plan MKTC.1/MKTC.3).
package billing

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/coupons"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// ImmediateRedeemInput creates a redeemed row without a prior reservation (100%-off free grant
// and webhook fallback when the reservation was swept — MKTC.3 FR-7, FR-10).
type ImmediateRedeemInput struct {
	CouponID          uuid.UUID
	UserID            uuid.UUID
	CourseID          uuid.UUID
	CoursePriceCents  int
	CourseCurrency    string
	AlreadyOwned      bool
	ProviderEventID   string
	EntitlementID     *uuid.UUID
	CheckoutSessionID *string
	// When set, skip re-evaluate and use these amounts (webhook fallback / honour reservation quote).
	ListPriceCents *int
	DiscountCents  *int
	ChargedCents   *int
	// SkipEligibility skips Evaluate (used when honouring a prior quote after archive).
	SkipEligibility bool
	Now             time.Time
}

// RedeemCouponImmediate inserts a redeemed row under the coupon row lock (idempotent on provider_event_id).
// Returns (redemption, reason, created, error). reason is non-ok when eligibility fails (no insert).
func RedeemCouponImmediate(ctx context.Context, pool *pgxpool.Pool, in ImmediateRedeemInput) (*Redemption, coupons.Reason, bool, error) {
	if in.ProviderEventID == "" {
		return nil, "", false, errors.New("provider_event_id required")
	}
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, "", false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Idempotent: existing row for this event.
	existing, err := scanRedemption(tx.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions WHERE provider_event_id = $1
`, in.ProviderEventID))
	if err == nil && existing != nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, "", false, err
		}
		telemetry.RecordCouponRedeem("idempotent")
		return existing, coupons.ReasonOK, false, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, "", false, err
	}

	c, err := scanCoupon(tx.QueryRow(ctx, `
SELECT `+couponSelectCols+`
FROM billing.course_coupons
WHERE id = $1
FOR UPDATE
`, in.CouponID))
	if errors.Is(err, pgx.ErrNoRows) {
		telemetry.RecordCouponRedeem("not_found")
		return nil, coupons.ReasonNotFound, false, nil
	}
	if err != nil {
		return nil, "", false, err
	}

	listCents := in.CoursePriceCents
	discountCents := 0
	chargedCents := in.CoursePriceCents
	currency := in.CourseCurrency
	if currency == "" {
		currency = "usd"
	}

	if !in.SkipEligibility {
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
			return nil, "", false, err
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
			return nil, "", false, err
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
			telemetry.RecordCouponRedeem(string(reason))
			return nil, reason, false, nil
		}
		listCents = quote.ListCents
		discountCents = quote.DiscountCents
		chargedCents = quote.ChargedCents
		currency = quote.Currency
	} else {
		if in.ListPriceCents != nil {
			listCents = *in.ListPriceCents
		}
		if in.DiscountCents != nil {
			discountCents = *in.DiscountCents
		}
		if in.ChargedCents != nil {
			chargedCents = *in.ChargedCents
		}
	}

	var session any
	if in.CheckoutSessionID != nil && *in.CheckoutSessionID != "" {
		session = *in.CheckoutSessionID
	}

	r, err := scanRedemption(tx.QueryRow(ctx, `
INSERT INTO billing.coupon_redemptions (
    coupon_id, course_id, user_id, entitlement_id, status,
    checkout_session_id, provider_event_id,
    list_price_cents, discount_cents, charged_cents, currency,
    reserved_at, redeemed_at, expires_at
) VALUES ($1,$2,$3,$4,'redeemed',$5,$6,$7,$8,$9,$10,$11,$11,NULL)
RETURNING `+redemptionSelectCols,
		c.ID, c.CourseID, in.UserID, in.EntitlementID, session, in.ProviderEventID,
		listCents, discountCents, chargedCents, currency, now,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Race on provider_event_id or session — re-read.
			existing, getErr := scanRedemption(tx.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions WHERE provider_event_id = $1
`, in.ProviderEventID))
			if getErr == nil {
				if err := tx.Commit(ctx); err != nil {
					return nil, "", false, err
				}
				telemetry.RecordCouponRedeem("idempotent")
				return existing, coupons.ReasonOK, false, nil
			}
		}
		return nil, "", false, err
	}

	if _, err := tx.Exec(ctx, `
UPDATE billing.course_coupons
SET redeemed_count = redeemed_count + 1, updated_at = NOW()
WHERE id = $1
`, c.ID); err != nil {
		return nil, "", false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", false, err
	}
	telemetry.RecordCouponRedeem("ok")
	return r, coupons.ReasonOK, true, nil
}

// LinkRedemptionEntitlement sets entitlement_id on a redeemed row if missing.
func LinkRedemptionEntitlement(ctx context.Context, pool *pgxpool.Pool, redemptionID, entitlementID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE billing.coupon_redemptions
SET entitlement_id = $2
WHERE id = $1 AND entitlement_id IS NULL
`, redemptionID, entitlementID)
	return err
}

// UpdateRedemptionChargedCents records the final charged amount from the provider (tax-ex subtotal when separable).
func UpdateRedemptionChargedCents(ctx context.Context, pool *pgxpool.Pool, redemptionID uuid.UUID, chargedCents int) error {
	_, err := pool.Exec(ctx, `
UPDATE billing.coupon_redemptions
SET charged_cents = $2
WHERE id = $1
`, redemptionID, chargedCents)
	return err
}
