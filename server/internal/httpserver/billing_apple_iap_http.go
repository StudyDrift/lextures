package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
)

func (d Deps) registerAppleIAPRoutes(r chi.Router) {
	r.Get("/api/v1/billing/apple/products", d.handleAppleIAPProducts())
	r.Post("/api/v1/billing/apple/verify", d.handleAppleIAPVerify())
}

func (d Deps) handleAppleIAPProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		if d.billingFeatureOff(w) {
			return
		}
		cfg := svcBilling.AppleIAPConfigFrom(d.effectiveConfig())
		var courseID *uuid.UUID
		if raw := strings.TrimSpace(r.URL.Query().Get("courseId")); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			courseID = &id
		}
		var products []svcBilling.AppleProductInfo
		if d.Pool != nil {
			list, err := svcBilling.ListAppleProductsForCourse(r.Context(), d.Pool, cfg, courseID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load Apple products.")
				return
			}
			products = list
		} else {
			// No DB (tests): subscription product ids from env only.
			products, _ = svcBilling.ListAppleProductsForCourse(r.Context(), nil, cfg, nil)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"configured": svcBilling.AppleIAPConfigured(cfg),
			"bundleId":   cfg.BundleID,
			"products":   products,
		})
	}
}

func (d Deps) handleAppleIAPVerify() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
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
		cfg := svcBilling.AppleIAPConfigFrom(d.effectiveConfig())
		if !svcBilling.AppleIAPConfigured(cfg) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Apple IAP is not configured.")
			return
		}
		var body struct {
			SignedTransaction string  `json:"signedTransaction"`
			CourseID          *string `json:"courseId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.SignedTransaction) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "signedTransaction is required.")
			return
		}
		var courseID *uuid.UUID
		if body.CourseID != nil && strings.TrimSpace(*body.CourseID) != "" {
			id, err := uuid.Parse(strings.TrimSpace(*body.CourseID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			courseID = &id
		}
		result, err := svcBilling.VerifyAndGrantAppleIAP(
			r.Context(), d.Pool, cfg, userID, body.SignedTransaction, courseID,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not verify Apple purchase.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(result)
	}
}
