package httpserver

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/learnerprogress"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// registerMarketplacePurchaseRoutes wires free-claim and paid-checkout endpoints (plan MKT4).
func (d Deps) registerMarketplacePurchaseRoutes(r chi.Router) {
	r.Post("/api/v1/marketplace/courses/{slug}/claim", d.handleMarketplaceClaim())
	r.Post("/api/v1/marketplace/courses/{slug}/checkout", d.handleMarketplaceCheckout())
}

type marketplaceClaimResponse struct {
	Enrolled      bool    `json:"enrolled"`
	EntitlementID string  `json:"entitlementId"`
	AlreadyOwned  bool    `json:"alreadyOwned,omitempty"`
	FirstItemID   *string `json:"firstItemId,omitempty"`
	CourseCode    string  `json:"courseCode"`
}

// handleMarketplaceClaim creates a free course_purchase entitlement and enrolls the learner
// (plan MKT4 FR-1, FR-5, FR-7, FR-11). Paid courses return 402 directing the client to checkout.
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
		if !repoCourse.IsFree(course.PriceCents) {
			telemetry.RecordMarketplaceClaim("payment_required")
			apierr.WritePaymentRequired(w, "Purchase required.", marketplaceCheckoutHint(course.Slug, course.CourseCode))
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
			d.writeMarketplaceClaimResponse(w, ctx, course, existing, true)
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
		d.writeMarketplaceClaimResponse(w, ctx, course, ent, !created)
	}
}

func (d Deps) writeMarketplaceClaimResponse(
	w http.ResponseWriter,
	ctx context.Context,
	course *repoCourse.MarketplaceCourse,
	ent *repoBilling.Entitlement,
	alreadyOwned bool,
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
	}
	if first != nil {
		s := first.String()
		resp.FirstItemID = &s
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleMarketplaceCheckout starts Stripe Checkout for a paid marketplace course (plan MKT4 FR-3).
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

		email, err := svcBilling.LookupUserEmail(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "User not found.")
			return
		}
		cfg := svcBilling.ConfigFrom(d.effectiveConfig())
		if !cfg.IsConfigured() {
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

		result, err := svcBilling.CreateCheckoutSession(r.Context(), d.Pool, cfg, svcBilling.CheckoutRequest{
			UserID:             userID,
			Email:              email,
			CourseID:           &courseID,
			SuccessURL:         successURL,
			CancelURL:          cancelURL,
			PlatformTaxEnabled: d.effectiveConfig().FFTaxCollection,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not start checkout.")
			return
		}
		telemetry.RecordMarketplaceCheckoutCreated()
		writeJSON(w, http.StatusOK, map[string]string{
			"sessionId":   result.SessionID,
			"checkoutUrl": result.CheckoutURL,
		})
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
