package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/learnerprogress"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
	"github.com/lextures/lextures/server/internal/service/coupons"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// registerMarketplacePurchaseRoutes wires free-claim, paid-checkout, and coupon preview (MKT4 / MKTC.3).
func (d Deps) registerMarketplacePurchaseRoutes(r chi.Router) {
	r.Post("/api/v1/marketplace/courses/{slug}/claim", d.handleMarketplaceClaim())
	r.Post("/api/v1/marketplace/courses/{slug}/checkout", d.handleMarketplaceCheckout())
	r.Post("/api/v1/marketplace/courses/{slug}/coupon/preview", d.handleMarketplaceCouponPreview())
}

type marketplaceClaimResponse struct {
	Enrolled      bool    `json:"enrolled"`
	EntitlementID string  `json:"entitlementId"`
	AlreadyOwned  bool    `json:"alreadyOwned,omitempty"`
	FirstItemID   *string `json:"firstItemId,omitempty"`
	CourseCode    string  `json:"courseCode"`
	GrantedFree   bool    `json:"grantedFree,omitempty"`
}

type marketplaceCouponBody struct {
	CouponCode string `json:"couponCode"`
	// Client-supplied amounts are intentionally ignored (FR-6).
	ChargedCents  *int `json:"chargedCents"`
	DiscountCents *int `json:"discountCents"`
	PriceCents    *int `json:"priceCents"`
}

// writeCouponApplyLimited writes 429 RATE_LIMITED with Retry-After and optional reason (MKTC.7).
func writeCouponApplyLimited(w http.ResponseWriter, reason svcBilling.CouponApplyLimitReason, retryAfter time.Duration) {
	secs := int(retryAfter.Round(time.Second) / time.Second)
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(secs))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusTooManyRequests)
	msg := "Coupon preview rate limit exceeded."
	clientReason := "rate_limited"
	if reason == svcBilling.CouponApplyLimitedCooldown {
		msg = "Too many failed coupon attempts. Try again in 15 minutes."
		clientReason = "cooldown"
		telemetry.RecordCouponApplyCooldown()
	} else {
		telemetry.RecordCouponApply("rate_limited")
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    apierr.CodeRateLimited,
			"message": msg,
		},
		"reason":     clientReason,
		"retryAfter": secs,
	})
}

// handleMarketplaceCouponPreview is POST .../coupon/preview (MKTC.3 + MKTC.7 abuse controls).
func (d Deps) handleMarketplaceCouponPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.couponsFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		course, err := repoCourse.GetMarketplaceCourseBySlug(r.Context(), d.Pool, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if course == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(course.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}

		ip := clientIP(r)
		lim := svcBilling.CheckCouponApplyLimit(userID, courseID, ip, time.Now())
		if !lim.Allowed {
			writeCouponApplyLimited(w, lim.Reason, lim.RetryAfter)
			return
		}

		var body struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		owned, _ := repoBilling.MarketplaceAccess(r.Context(), d.Pool, userID, courseID)
		prev, err := svcBilling.PreviewCoupon(r.Context(), d.Pool, svcBilling.PreviewCouponInput{
			CourseID:       courseID,
			UserID:         userID,
			Code:           body.Code,
			CoursePrice:    course.PriceCents,
			CourseCurrency: course.PriceCurrency,
			AlreadyOwned:   owned,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to evaluate coupon.")
			return
		}

		success := prev != nil && prev.Applied
		svcBilling.RecordCouponApplyOutcome(userID, courseID, success, time.Now())
		if !success && prev != nil {
			// FR-3: log failed applies; raw code never stored for not_found.
			codeHash := svcBilling.HashCouponAttemptCode(prev.Code)
			if prev.Reason == coupons.ReasonNotFound {
				codeHash = svcBilling.HashCouponAttemptCode(body.Code)
			}
			_ = repoBilling.InsertCouponAttempt(r.Context(), d.Pool, repoBilling.InsertCouponAttemptInput{
				UserID:   userID,
				CourseID: courseID,
				CodeHash: codeHash,
				Reason:   string(prev.Reason),
				IPPrefix: svcBilling.IPPrefix(ip),
			})
		}
		writeJSON(w, http.StatusOK, prev)
	}
}

// handleMarketplaceClaim creates a free course_purchase entitlement and enrolls the learner
// (plan MKT4 FR-1, FR-5, FR-7, FR-11). Paid courses return 402 unless a 100%-off coupon is supplied (MKTC.3 FR-8).
func (d Deps) handleMarketplaceClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.courseMarketplaceOff(w) {
			return
		}
		if !d.checkBillingCheckoutRateLimit(userID) {
			telemetry.RecordMarketplaceClaim("rate_limited")
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Checkout rate limit exceeded.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		course, err := repoCourse.GetMarketplaceCourseBySlug(r.Context(), d.Pool, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if course == nil {
			telemetry.RecordMarketplaceClaim("not_found")
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(course.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}

		couponCode := ""
		{
			var body marketplaceCouponBody
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				couponCode = strings.TrimSpace(body.CouponCode)
			}
		}
		// Optional coupon on paid course (FR-8).
		if !repoCourse.IsFree(course.PriceCents) {
			if couponCode == "" || !d.effectiveConfig().FFCourseCoupons {
				telemetry.RecordMarketplaceClaim("payment_required")
				apierr.WritePaymentRequired(w, "Purchase required.", marketplaceCheckoutHint(course.Slug, course.CourseCode))
				return
			}
			d.handleCouponFreeGrant(w, r, userID, course, courseID, couponCode)
			return
		}

		ctx := r.Context()
		existing, err := repoBilling.ActiveCoursePurchase(ctx, d.Pool, userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check ownership.")
			return
		}
		if existing != nil {
			telemetry.RecordMarketplaceClaim("already_owned")
			_, _ = courseroles.EnrollStudentWithGrants(ctx, d.Pool, courseID, userID, course.CourseCode)
			d.writeMarketplaceClaimResponse(w, ctx, course, existing, true, false)
			return
		}

		ent, created, err := repoBilling.CreateCourseGrantIdempotent(ctx, d.Pool, repoBilling.CourseGrantInput{
			UserID:            userID,
			CourseID:          courseID,
			AcquisitionSource: repoBilling.AcquisitionFree,
			AmountPaidCents:   0,
			Currency:          course.PriceCurrency,
		})
		if err != nil || ent == nil {
			telemetry.RecordMarketplaceClaim("error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create entitlement.")
			return
		}

		enrolledNew, err := courseroles.EnrollStudentWithGrants(ctx, d.Pool, courseID, userID, course.CourseCode)
		if err != nil {
			telemetry.RecordMarketplaceClaim("error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enroll.")
			return
		}
		if enrolledNew {
			d.notifyCourses(userID)
		}

		if created {
			telemetry.RecordMarketplaceClaim("created")
		} else {
			telemetry.RecordMarketplaceClaim("already_owned")
		}
		d.writeMarketplaceClaimResponse(w, ctx, course, ent, !created, false)
	}
}

func (d Deps) handleCouponFreeGrant(
	w http.ResponseWriter,
	r *http.Request,
	userID uuid.UUID,
	course *repoCourse.MarketplaceCourse,
	courseID uuid.UUID,
	couponCode string,
) {
	ctx := r.Context()
	owned, err := repoBilling.MarketplaceAccess(ctx, d.Pool, userID, courseID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check ownership.")
		return
	}
	if owned {
		existing, _ := repoBilling.ActiveCoursePurchase(ctx, d.Pool, userID, courseID)
		if existing != nil {
			_, _ = courseroles.EnrollStudentWithGrants(ctx, d.Pool, courseID, userID, course.CourseCode)
			d.writeMarketplaceClaimResponse(w, ctx, course, existing, true, false)
			return
		}
	}

	result, err := svcBilling.GrantFreeWithCoupon(ctx, d.Pool, svcBilling.FreeGrantInput{
		UserID:         userID,
		CourseID:       courseID,
		CourseCode:     course.CourseCode,
		Code:           couponCode,
		CoursePrice:    course.PriceCents,
		CourseCurrency: course.PriceCurrency,
		AlreadyOwned:   owned,
	})
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to redeem coupon.")
		return
	}
	if result.Reason != coupons.ReasonOK {
		writeCouponRejected(w, result.Reason, course.PriceCents, course.PriceCurrency)
		return
	}
	if result.Entitlement == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create entitlement.")
		return
	}
	enrolledNew, err := courseroles.EnrollStudentWithGrants(ctx, d.Pool, courseID, userID, course.CourseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enroll.")
		return
	}
	if enrolledNew {
		d.notifyCourses(userID)
	}
	telemetry.RecordMarketplaceClaim("created")
	d.writeMarketplaceClaimResponse(w, ctx, course, result.Entitlement, !result.Created, true)
}

func (d Deps) writeMarketplaceClaimResponse(
	w http.ResponseWriter,
	ctx context.Context,
	course *repoCourse.MarketplaceCourse,
	ent *repoBilling.Entitlement,
	alreadyOwned bool,
	grantedFree bool,
) {
	courseID, _ := uuid.Parse(course.ID)
	first, err := learnerprogress.FirstItem(ctx, d.Pool, courseID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course content.")
		return
	}
	resp := marketplaceClaimResponse{
		Enrolled:      true,
		EntitlementID: ent.ID.String(),
		AlreadyOwned:  alreadyOwned,
		CourseCode:    course.CourseCode,
		GrantedFree:   grantedFree,
	}
	if first != nil {
		s := first.String()
		resp.FirstItemID = &s
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleMarketplaceCheckout starts Stripe Checkout for a paid marketplace course (plan MKT4 FR-3 / MKTC.3 FR-4).
func (d Deps) handleMarketplaceCheckout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.courseMarketplaceOff(w) {
			return
		}
		if d.billingFeatureOff(w) {
			return
		}
		if !d.checkBillingCheckoutRateLimit(userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Checkout rate limit exceeded.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		slug := strings.TrimSpace(chi.URLParam(r, "slug"))
		course, err := repoCourse.GetMarketplaceCourseBySlug(r.Context(), d.Pool, slug)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if course == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		courseID, err := uuid.Parse(course.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
			return
		}
		if repoCourse.IsFree(course.PriceCents) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This course is free. Use the claim endpoint instead.")
			return
		}

		// Optional couponCode body (FR-4). Client amounts are ignored (FR-6).
		couponCode := ""
		{
			var body marketplaceCouponBody
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				couponCode = strings.TrimSpace(body.CouponCode)
			}
		}
		// Flag off: ignore couponCode so behaviour is unchanged (rollout FR).
		if !d.effectiveConfig().FFCourseCoupons {
			couponCode = ""
		}

		owned, err := repoBilling.MarketplaceAccess(r.Context(), d.Pool, userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check ownership.")
			return
		}
		if owned {
			writeJSON(w, http.StatusOK, map[string]any{
				"alreadyOwned": true,
				"courseCode":   course.CourseCode,
				"courseId":     course.ID,
			})
			return
		}

		// Coupon path.
		var reserved *svcBilling.CheckoutCouponResult
		if couponCode != "" {
			reserved, err = svcBilling.ResolveAndReserveCoupon(r.Context(), d.Pool, svcBilling.PreviewCouponInput{
				CourseID:       courseID,
				UserID:         userID,
				Code:           couponCode,
				CoursePrice:    course.PriceCents,
				CourseCurrency: course.PriceCurrency,
				AlreadyOwned:   false,
			})
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to apply coupon.")
				return
			}
			if reserved.Reason != coupons.ReasonOK {
				writeCouponRejected(w, reserved.Reason, course.PriceCents, course.PriceCurrency)
				return
			}
			// Free after discount → grant path (FR-7). Existing reservation is promoted inside GrantFreeWithCoupon.
			if reserved.Quote.ChargedCents == 0 {
				d.handleCouponFreeGrant(w, r, userID, course, courseID, couponCode)
				return
			}
		}

		email, err := svcBilling.LookupUserEmail(r.Context(), d.Pool, userID)
		if err != nil {
			if reserved != nil && reserved.Redemption != nil {
				_ = repoBilling.ReleaseCouponReservation(r.Context(), d.Pool, reserved.Redemption.ID, "checkout_error")
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "User not found.")
			return
		}
		cfg := svcBilling.ConfigFrom(d.effectiveConfig())
		if !cfg.IsConfigured() {
			if reserved != nil && reserved.Redemption != nil {
				_ = repoBilling.ReleaseCouponReservation(r.Context(), d.Pool, reserved.Redemption.ID, "checkout_error")
			}
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Stripe is not configured.")
			return
		}

		origin := strings.TrimRight(strings.TrimSpace(d.effectiveConfig().PublicWebOrigin), "/")
		detailSlug := course.Slug
		if detailSlug == "" {
			detailSlug = course.CourseCode
		}
		successURL := origin + "/checkout/success?course_id=" + url.QueryEscape(course.ID) +
			"&course_code=" + url.QueryEscape(course.CourseCode) +
			"&slug=" + url.QueryEscape(detailSlug)
		cancelURL := origin + "/checkout/cancel?slug=" + url.QueryEscape(detailSlug)
		if couponCode != "" {
			norm := coupons.NormalizeCode(couponCode)
			cancelURL += "&coupon=" + url.QueryEscape(norm)
			// MKTC.5 FR-15: success page shows code + savings from URL params.
			successURL += "&coupon=" + url.QueryEscape(norm)
			if reserved != nil {
				successURL += "&discount_cents=" + url.QueryEscape(strconv.Itoa(reserved.Quote.DiscountCents))
				successURL += "&currency=" + url.QueryEscape(strings.ToLower(strings.TrimSpace(reserved.Quote.Currency)))
			}
		}

		checkoutReq := svcBilling.CheckoutRequest{
			UserID:             userID,
			Email:              email,
			CourseID:           &courseID,
			SuccessURL:         successURL,
			CancelURL:          cancelURL,
			PlatformTaxEnabled: d.effectiveConfig().FFTaxCollection,
		}
		if reserved != nil && reserved.Coupon != nil && reserved.Redemption != nil {
			checkoutReq.HasFirstPartyCoupon = true
			checkoutReq.DiscountCents = reserved.Quote.DiscountCents
			checkoutReq.ChargedCents = reserved.Quote.ChargedCents
			checkoutReq.CouponID = reserved.Coupon.ID.String()
			checkoutReq.CouponCode = reserved.Coupon.Code
			checkoutReq.ListPriceCents = reserved.Quote.ListCents
		}

		result, err := svcBilling.CreateCheckoutSession(r.Context(), d.Pool, cfg, checkoutReq)
		if err != nil {
			if reserved != nil && reserved.Redemption != nil {
				_ = repoBilling.ReleaseCouponReservation(r.Context(), d.Pool, reserved.Redemption.ID, "provider_error")
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not start checkout.")
			return
		}
		if reserved != nil && reserved.Redemption != nil {
			if err := svcBilling.AttachCheckoutSession(r.Context(), d.Pool, reserved.Redemption.ID, result.SessionID); err != nil {
				// Session exists; leave reservation with TTL. Best-effort attach failure is non-fatal for redirect.
				_ = err
			}
			telemetry.RecordCouponCheckoutCreated(true)
		}
		telemetry.RecordMarketplaceCheckoutCreated()
		resp := map[string]any{
			"sessionId":   result.SessionID,
			"checkoutUrl": result.CheckoutURL,
		}
		if reserved != nil {
			resp["chargedCents"] = reserved.Quote.ChargedCents
			resp["currency"] = reserved.Quote.Currency
		} else {
			resp["chargedCents"] = course.PriceCents
			resp["currency"] = course.PriceCurrency
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func writeCouponRejected(w http.ResponseWriter, reason coupons.Reason, listPriceCents int, currency string) {
	curr := strings.ToLower(strings.TrimSpace(currency))
	if curr == "" {
		curr = "usd"
	}
	msg := couponReasonMessage(reason)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    string(apierr.CodeUnprocessableEntity),
			"message": msg,
		},
		"reason":         reason,
		"listPriceCents": listPriceCents,
		"currency":       curr,
	})
}

func couponReasonMessage(reason coupons.Reason) string {
	switch reason {
	case coupons.ReasonNotFound:
		return "That coupon code was not found."
	case coupons.ReasonInactive:
		return "That coupon code is no longer active."
	case coupons.ReasonNotStarted:
		return "That coupon code is not active yet."
	case coupons.ReasonExpired:
		return "That coupon code has expired."
	case coupons.ReasonExhausted:
		return "That coupon code has no seats left."
	case coupons.ReasonAlreadyUsed:
		return "You have already used that coupon code."
	case coupons.ReasonCurrencyMismatch:
		return "That coupon code cannot be used with this course currency."
	case coupons.ReasonCourseFree:
		return "This course is already free."
	case coupons.ReasonOwned:
		return "You already own this course."
	default:
		return "That coupon code cannot be applied."
	}
}

func marketplaceCheckoutHint(slug, courseCode string) string {
	s := strings.TrimSpace(slug)
	if s == "" {
		s = strings.TrimSpace(courseCode)
	}
	if s == "" {
		return "/marketplace"
	}
	return "/marketplace/" + s
}
