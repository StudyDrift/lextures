// Coupon-aware marketplace preview, reservation, and free-grant (plan MKTC.3).
package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/service/coupons"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// SeatsRemainingDisclosureThreshold is the max remaining seats still shown to learners
// (MKTC.3 §18 Q3). Above this, seatsRemaining is null so cap state is not leaked.
const SeatsRemainingDisclosureThreshold = 10

// CouponPreview is the learner-facing preview / detail coupon breakdown (FR-1, FR-18).
type CouponPreview struct {
	Applied           bool           `json:"applied"`
	Code              string         `json:"code"`
	Reason            coupons.Reason `json:"reason"`
	ListPriceCents    int            `json:"listPriceCents"`
	DiscountCents     int            `json:"discountCents"`
	ChargedCents      int            `json:"chargedCents"`
	Currency          string         `json:"currency"`
	FreeAfterDiscount bool           `json:"freeAfterDiscount"`
	EndsAt            *string        `json:"endsAt"`
	SeatsRemaining    *int           `json:"seatsRemaining"`
	ClampedToFree     bool           `json:"clampedToFree,omitempty"`
}

// PreviewCouponInput is the pure-read preview payload (no reservation).
type PreviewCouponInput struct {
	CourseID       uuid.UUID
	UserID         uuid.UUID
	Code           string
	CoursePrice    int
	CourseCurrency string
	AlreadyOwned   bool
	Now            time.Time
}

// PreviewCoupon evaluates a code against a course without creating a reservation (FR-1, FR-2).
func PreviewCoupon(ctx context.Context, pool *pgxpool.Pool, in PreviewCouponInput) (*CouponPreview, error) {
	code := coupons.NormalizeCode(in.Code)
	list := in.CoursePrice
	curr := strings.ToLower(strings.TrimSpace(in.CourseCurrency))
	if curr == "" {
		curr = "usd"
	}
	base := &CouponPreview{
		Applied:        false,
		Code:           code,
		Reason:         coupons.ReasonNotFound,
		ListPriceCents: list,
		DiscountCents:  0,
		ChargedCents:   list,
		Currency:       curr,
	}
	if code == "" {
		telemetry.RecordCouponApply(string(coupons.ReasonNotFound))
		return base, nil
	}

	c, err := repoBilling.GetCouponByCourseAndCode(ctx, pool, in.CourseID, code)
	if err != nil {
		return nil, err
	}
	if c == nil {
		telemetry.RecordCouponApply(string(coupons.ReasonNotFound))
		return base, nil
	}
	base.Code = c.Code

	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	consumed := 0
	userSeats := 0
	if counts, err := repoBilling.CouponSeatCounts(ctx, pool, []uuid.UUID{c.ID}); err == nil {
		if sc, ok := counts[c.ID]; ok {
			consumed = sc.Consumed
		}
	} else {
		return nil, err
	}
	if n, err := repoBilling.UserSeatCount(ctx, pool, c.ID, in.UserID); err == nil {
		userSeats = n
	} else {
		return nil, err
	}

	domain := c.ToDomain()
	reason, quote := coupons.Evaluate(&domain, coupons.EvalContext{
		Now:            now,
		CoursePrice:    list,
		CourseCurrency: curr,
		ConsumedSeats:  consumed,
		UserSeats:      userSeats,
		AlreadyOwned:   in.AlreadyOwned,
	})

	out := &CouponPreview{
		Applied:           reason == coupons.ReasonOK,
		Code:              c.Code,
		Reason:            reason,
		ListPriceCents:    quote.ListCents,
		DiscountCents:     quote.DiscountCents,
		ChargedCents:      quote.ChargedCents,
		Currency:          quote.Currency,
		FreeAfterDiscount: quote.ChargedCents == 0 && reason == coupons.ReasonOK,
		ClampedToFree:     quote.ClampedToFree,
	}
	if reason != coupons.ReasonOK {
		// FR-1: failure still returns list as charged.
		out.ListPriceCents = list
		out.DiscountCents = 0
		out.ChargedCents = list
		out.Currency = curr
	}
	if c.EndsAt != nil {
		s := c.EndsAt.UTC().Format(time.RFC3339)
		out.EndsAt = &s
	}
	out.SeatsRemaining = seatsRemainingDisclosure(c.MaxRedemptions, consumed)
	telemetry.RecordCouponApply(string(reason))
	if quote.ClampedToFree {
		telemetry.RecordCouponClampedToFree()
	}
	return out, nil
}

func seatsRemainingDisclosure(max *int, consumed int) *int {
	if max == nil {
		return nil
	}
	rem := *max - consumed
	if rem < 0 {
		rem = 0
	}
	if rem > SeatsRemainingDisclosureThreshold {
		return nil
	}
	return &rem
}

// CheckoutCouponResult is the outcome of resolving + reserving a coupon for paid checkout.
type CheckoutCouponResult struct {
	Preview     *CouponPreview
	Coupon      *repoBilling.Coupon
	Redemption  *repoBilling.Redemption
	Quote       coupons.Quote
	Reason      coupons.Reason
	ClampedFree bool
}

// ResolveAndReserveCoupon re-evaluates eligibility and reserves a seat (FR-4).
// When charged is 0 the caller must take the free-grant path instead (no provider).
func ResolveAndReserveCoupon(ctx context.Context, pool *pgxpool.Pool, in PreviewCouponInput) (*CheckoutCouponResult, error) {
	code := coupons.NormalizeCode(in.Code)
	if code == "" {
		return &CheckoutCouponResult{Reason: coupons.ReasonNotFound, Preview: emptyPreview(code, in)}, nil
	}
	c, err := repoBilling.GetCouponByCourseAndCode(ctx, pool, in.CourseID, code)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return &CheckoutCouponResult{Reason: coupons.ReasonNotFound, Preview: emptyPreview(code, in)}, nil
	}

	// Reuse an active reservation for this user (retry-safe).
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if existing, err := repoBilling.GetActiveReservationForUser(ctx, pool, c.ID, in.UserID, now); err != nil {
		return nil, err
	} else if existing != nil {
		q := coupons.Quote{
			ListCents:     existing.ListPriceCents,
			DiscountCents: existing.DiscountCents,
			ChargedCents:  existing.ChargedCents,
			Currency:      existing.Currency,
		}
		prev := previewFromQuote(c, q, coupons.ReasonOK, 0)
		return &CheckoutCouponResult{
			Preview:    prev,
			Coupon:     c,
			Redemption: existing,
			Quote:      q,
			Reason:     coupons.ReasonOK,
		}, nil
	}

	res, reason, err := repoBilling.ReserveCoupon(ctx, pool, repoBilling.ReserveInput{
		CouponID:         c.ID,
		UserID:           in.UserID,
		CoursePriceCents: in.CoursePrice,
		CourseCurrency:   in.CourseCurrency,
		AlreadyOwned:     in.AlreadyOwned,
		Now:              now,
	})
	if err != nil {
		return nil, err
	}
	if reason != coupons.ReasonOK {
		prev, perr := PreviewCoupon(ctx, pool, in)
		if perr != nil {
			prev = emptyPreview(c.Code, in)
			prev.Reason = reason
		}
		return &CheckoutCouponResult{Reason: reason, Preview: prev, Coupon: c}, nil
	}
	q := coupons.Quote{
		ListCents:     res.ListPriceCents,
		DiscountCents: res.DiscountCents,
		ChargedCents:  res.ChargedCents,
		Currency:      res.Currency,
	}
	// Detect clamp from zero charged with positive list.
	clamped := q.ChargedCents == 0 && q.ListCents > 0 && q.DiscountCents == q.ListCents
	if domain := c.ToDomain(); domain.Kind == coupons.KindPercent || domain.Kind == coupons.KindFixed {
		// Re-apply for ClampedToFree flag (display/telemetry).
		full := coupons.ApplyDiscount(in.CoursePrice, in.CourseCurrency, domain)
		clamped = full.ClampedToFree
		if clamped {
			telemetry.RecordCouponClampedToFree()
		}
	}
	prev := previewFromQuote(c, q, coupons.ReasonOK, 0)
	prev.ClampedToFree = clamped
	prev.FreeAfterDiscount = q.ChargedCents == 0
	return &CheckoutCouponResult{
		Preview:     prev,
		Coupon:      c,
		Redemption:  res,
		Quote:       q,
		Reason:      coupons.ReasonOK,
		ClampedFree: clamped,
	}, nil
}

func emptyPreview(code string, in PreviewCouponInput) *CouponPreview {
	curr := strings.ToLower(strings.TrimSpace(in.CourseCurrency))
	if curr == "" {
		curr = "usd"
	}
	return &CouponPreview{
		Applied:        false,
		Code:           code,
		Reason:         coupons.ReasonNotFound,
		ListPriceCents: in.CoursePrice,
		DiscountCents:  0,
		ChargedCents:   in.CoursePrice,
		Currency:       curr,
	}
}

func previewFromQuote(c *repoBilling.Coupon, q coupons.Quote, reason coupons.Reason, consumed int) *CouponPreview {
	out := &CouponPreview{
		Applied:           reason == coupons.ReasonOK,
		Code:              c.Code,
		Reason:            reason,
		ListPriceCents:    q.ListCents,
		DiscountCents:     q.DiscountCents,
		ChargedCents:      q.ChargedCents,
		Currency:          q.Currency,
		FreeAfterDiscount: q.ChargedCents == 0 && reason == coupons.ReasonOK,
		ClampedToFree:     q.ClampedToFree,
	}
	if c.EndsAt != nil {
		s := c.EndsAt.UTC().Format(time.RFC3339)
		out.EndsAt = &s
	}
	out.SeatsRemaining = seatsRemainingDisclosure(c.MaxRedemptions, consumed)
	return out
}

// FreeGrantInput is the 100%-off / clamp-to-free checkout path (FR-7, FR-8).
type FreeGrantInput struct {
	UserID         uuid.UUID
	CourseID       uuid.UUID
	CourseCode     string
	Code           string
	CoursePrice    int
	CourseCurrency string
	AlreadyOwned   bool
}

// FreeGrantResult is returned when a coupon makes the course free.
type FreeGrantResult struct {
	Entitlement *repoBilling.Entitlement
	Redemption  *repoBilling.Redemption
	Created     bool
	Reason      coupons.Reason
	Preview     *CouponPreview
}

// GrantFreeWithCoupon creates a coupon entitlement + redeemed ledger row without a payment provider.
func GrantFreeWithCoupon(ctx context.Context, pool *pgxpool.Pool, in FreeGrantInput) (*FreeGrantResult, error) {
	code := coupons.NormalizeCode(in.Code)
	if code == "" {
		return &FreeGrantResult{Reason: coupons.ReasonNotFound, Preview: emptyPreview(code, PreviewCouponInput{
			CoursePrice: in.CoursePrice, CourseCurrency: in.CourseCurrency,
		})}, nil
	}
	c, err := repoBilling.GetCouponByCourseAndCode(ctx, pool, in.CourseID, code)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return &FreeGrantResult{Reason: coupons.ReasonNotFound, Preview: emptyPreview(code, PreviewCouponInput{
			CoursePrice: in.CoursePrice, CourseCurrency: in.CourseCurrency,
		})}, nil
	}

	// If user already has an active reservation from a prior paid attempt that became free, release path is not needed —
	// free grant uses immediate redeem. Clear any open reservation first so seat math is clean.
	now := time.Now().UTC()
	if existing, err := repoBilling.GetActiveReservationForUser(ctx, pool, c.ID, in.UserID, now); err != nil {
		return nil, err
	} else if existing != nil {
		// Promote the reservation instead of double-counting.
		eventID := fmt.Sprintf("coupon_free:%s", existing.ID.String())
		ent, created, err := repoBilling.CreateCourseGrantIdempotent(ctx, pool, repoBilling.CourseGrantInput{
			UserID:            in.UserID,
			CourseID:          in.CourseID,
			AcquisitionSource: repoBilling.AcquisitionCoupon,
			AmountPaidCents:   0,
			Currency:          existing.Currency,
			StripeEventID:     &eventID,
		})
		if err != nil || ent == nil {
			return nil, fmt.Errorf("create coupon grant: %w", err)
		}
		var entID *uuid.UUID
		if ent != nil {
			id := ent.ID
			entID = &id
		}
		red, _, err := repoBilling.RedeemCoupon(ctx, pool, repoBilling.RedeemInput{
			RedemptionID:    existing.ID,
			ProviderEventID: eventID,
			EntitlementID:   entID,
		})
		if err != nil {
			return nil, err
		}
		telemetry.RecordCouponFreeGrant()
		if red != nil && red.DiscountCents > 0 {
			telemetry.RecordCouponDiscountCents(red.DiscountCents)
		}
		telemetry.RecordCouponRedeemed()
		return &FreeGrantResult{
			Entitlement: ent,
			Redemption:  red,
			Created:     created,
			Reason:      coupons.ReasonOK,
			Preview:     previewFromQuote(c, coupons.Quote{ListCents: existing.ListPriceCents, DiscountCents: existing.DiscountCents, ChargedCents: 0, Currency: existing.Currency}, coupons.ReasonOK, 0),
		}, nil
	}

	eventID := fmt.Sprintf("coupon_free:%s:%s:%s", in.UserID.String(), in.CourseID.String(), c.ID.String())
	// Create entitlement first (idempotent), then redeem ledger.
	ent, created, err := repoBilling.CreateCourseGrantIdempotent(ctx, pool, repoBilling.CourseGrantInput{
		UserID:            in.UserID,
		CourseID:          in.CourseID,
		AcquisitionSource: repoBilling.AcquisitionCoupon,
		AmountPaidCents:   0,
		Currency:          in.CourseCurrency,
		StripeEventID:     &eventID,
	})
	if err != nil || ent == nil {
		return nil, fmt.Errorf("create coupon grant: %w", err)
	}
	var entID *uuid.UUID
	if ent != nil {
		id := ent.ID
		entID = &id
	}
	// Ownership is short-circuited by the HTTP layer; pass false so Evaluate does not
	// reject a concurrent first-time free grant as "owned".
	red, reason, redCreated, err := repoBilling.RedeemCouponImmediate(ctx, pool, repoBilling.ImmediateRedeemInput{
		CouponID:         c.ID,
		UserID:           in.UserID,
		CourseID:         in.CourseID,
		CoursePriceCents: in.CoursePrice,
		CourseCurrency:   in.CourseCurrency,
		AlreadyOwned:     false,
		ProviderEventID:  eventID,
		EntitlementID:    entID,
	})
	if err != nil {
		return nil, err
	}
	if reason != coupons.ReasonOK {
		// Entitlement may already exist from a prior attempt; leave it (idempotent ownership).
		prev, _ := PreviewCoupon(ctx, pool, PreviewCouponInput{
			CourseID: in.CourseID, UserID: in.UserID, Code: code,
			CoursePrice: in.CoursePrice, CourseCurrency: in.CourseCurrency, AlreadyOwned: in.AlreadyOwned,
		})
		return &FreeGrantResult{Reason: reason, Preview: prev, Entitlement: ent, Created: created}, nil
	}
	if redCreated {
		telemetry.RecordCouponFreeGrant()
		if red != nil && red.DiscountCents > 0 {
			telemetry.RecordCouponDiscountCents(red.DiscountCents)
		}
		telemetry.RecordCouponRedeemed()
	}
	prev := previewFromQuote(c, coupons.Quote{
		ListCents: red.ListPriceCents, DiscountCents: red.DiscountCents,
		ChargedCents: 0, Currency: red.Currency,
	}, coupons.ReasonOK, 0)
	prev.FreeAfterDiscount = true
	return &FreeGrantResult{
		Entitlement: ent,
		Redemption:  red,
		Created:     created,
		Reason:      coupons.ReasonOK,
		Preview:     prev,
	}, nil
}

// AttachCheckoutSession records the provider session id on the reservation (FR-4e).
func AttachCheckoutSession(ctx context.Context, pool *pgxpool.Pool, redemptionID uuid.UUID, sessionID string) error {
	return repoBilling.SetRedemptionCheckoutSession(ctx, pool, redemptionID, sessionID)
}

// PromoteCouponRedemptionFromWebhook promotes a reservation (or creates a redeemed row) after payment (FR-10).
func PromoteCouponRedemptionFromWebhook(
	ctx context.Context,
	pool *pgxpool.Pool,
	sessionID string,
	eventID string,
	entitlementID *uuid.UUID,
	meta map[string]string,
	chargedCents int,
) error {
	if strings.TrimSpace(eventID) == "" {
		return nil
	}
	// Prefer existing reservation by session.
	if sessionID != "" {
		red, created, err := repoBilling.RedeemCoupon(ctx, pool, repoBilling.RedeemInput{
			CheckoutSessionID: sessionID,
			ProviderEventID:   eventID,
			EntitlementID:     entitlementID,
		})
		if err == nil && red != nil {
			if chargedCents > 0 {
				_ = repoBilling.UpdateRedemptionChargedCents(ctx, pool, red.ID, chargedCents)
			}
			// FR-10: link entitlement when redeem raced ahead of entitlement creation.
			if entitlementID != nil && red.EntitlementID == nil {
				_ = repoBilling.LinkRedemptionEntitlement(ctx, pool, red.ID, *entitlementID)
			}
			if created {
				telemetry.RecordCouponRedeemed()
				if red.DiscountCents > 0 {
					telemetry.RecordCouponDiscountCents(red.DiscountCents)
				}
			}
			return nil
		}
		// pgx.ErrNoRows or other — fall through to direct redeem when coupon_id present.
		if err != nil && !isNoRows(err) {
			// Already redeemed with different event? treat as ok if we can load by session.
			if existing, getErr := repoBilling.GetRedemptionByCheckoutSession(ctx, pool, sessionID); getErr == nil && existing != nil && existing.Status == repoBilling.RedemptionRedeemed {
				return nil
			}
			// Continue to fallback for not-found; return real errors otherwise only if no coupon meta.
		}
	}

	couponIDRaw := strings.TrimSpace(meta["coupon_id"])
	if couponIDRaw == "" {
		return nil
	}
	couponID, err := uuid.Parse(couponIDRaw)
	if err != nil {
		return nil
	}
	userID, err := uuid.Parse(strings.TrimSpace(meta["user_id"]))
	if err != nil {
		return nil
	}
	courseID, err := uuid.Parse(strings.TrimSpace(meta["course_id"]))
	if err != nil {
		return nil
	}
	listCents := atoiDefault(meta["list_price_cents"], 0)
	discountCents := atoiDefault(meta["coupon_discount_cents"], 0)
	if chargedCents <= 0 {
		chargedCents = atoiDefault(meta["charged_cents"], listCents-discountCents)
	}
	sess := sessionID
	var sessPtr *string
	if sess != "" {
		sessPtr = &sess
	}
	red, reason, created, err := repoBilling.RedeemCouponImmediate(ctx, pool, repoBilling.ImmediateRedeemInput{
		CouponID:          couponID,
		UserID:            userID,
		CourseID:          courseID,
		CoursePriceCents:  listCents,
		CourseCurrency:    strings.ToLower(strings.TrimSpace(meta["currency"])),
		ProviderEventID:   eventID,
		EntitlementID:     entitlementID,
		CheckoutSessionID: sessPtr,
		ListPriceCents:    &listCents,
		DiscountCents:     &discountCents,
		ChargedCents:      &chargedCents,
		SkipEligibility:   true, // FR-16 / FR-10: honour completed payment even if coupon later archived
	})
	if err != nil {
		return err
	}
	if reason != coupons.ReasonOK && red == nil {
		return nil
	}
	if created {
		telemetry.RecordCouponRedeemed()
		if discountCents > 0 {
			telemetry.RecordCouponDiscountCents(discountCents)
		}
	}
	return nil
}

// ReleaseCouponOnRefund marks a redeemed coupon seat as released (FR-11).
func ReleaseCouponOnRefund(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) error {
	red, err := repoBilling.GetRedeemedByUserAndCourse(ctx, pool, userID, courseID)
	if err != nil || red == nil {
		return err
	}
	return repoBilling.ReleaseCouponReservation(ctx, pool, red.ID, "refund")
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func atoiDefault(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
