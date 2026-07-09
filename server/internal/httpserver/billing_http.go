package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/notificationevents"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
	"github.com/lextures/lextures/server/internal/service/notifications"
	"github.com/lextures/lextures/server/internal/service/paymentprovider"
	repoPayments "github.com/lextures/lextures/server/internal/repos/payments"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const billingCheckoutRateLimitPerMinute = 10

var (
	billingRateMu    sync.Mutex
	billingRateByUID = map[uuid.UUID]billingRateEntry{}
)

type billingRateEntry struct {
	windowStart time.Time
	count       int
}

func (d Deps) billingFeatureOff(w http.ResponseWriter) bool {
	return d.paymentsFeatureOff(w)
}

func (d Deps) checkBillingCheckoutRateLimit(userID uuid.UUID) bool {
	billingRateMu.Lock()
	defer billingRateMu.Unlock()
	now := time.Now()
	e, ok := billingRateByUID[userID]
	if !ok || now.Sub(e.windowStart) >= time.Minute {
		billingRateByUID[userID] = billingRateEntry{windowStart: now, count: 1}
		return true
	}
	if e.count >= billingCheckoutRateLimitPerMinute {
		return false
	}
	e.count++
	billingRateByUID[userID] = e
	return true
}

func (d Deps) registerBillingRoutes(r chi.Router) {
	r.Post("/api/v1/billing/checkout", d.handleBillingCheckout())
	r.Get("/api/v1/billing/portal", d.handleBillingPortal())
	r.Get("/api/v1/me/entitlements", d.handleMyEntitlements())
	r.Get("/api/v1/me/purchases", d.handleMyPurchases())
	r.Get("/api/v1/internal/entitlements/check", d.handleInternalEntitlementCheck())
	r.Post("/api/v1/webhooks/stripe", d.handleStripeWebhook())
}

type entitlementJSON struct {
	ID              string   `json:"id"`
	EntitlementType string   `json:"entitlementType"`
	CourseID        *string  `json:"courseId,omitempty"`
	AmountPaidCents int      `json:"amountPaidCents"`
	SubtotalCents   int      `json:"subtotalCents,omitempty"`
	TaxAmountCents  int      `json:"taxAmountCents,omitempty"`
	TaxType         string   `json:"taxType,omitempty"`
	TaxJurisdiction string   `json:"taxJurisdiction,omitempty"`
	ReverseCharge   bool     `json:"reverseCharge,omitempty"`
	InvoiceID       *string  `json:"invoiceId,omitempty"`
	Currency        string   `json:"currency"`
	ValidFrom       string   `json:"validFrom"`
	ValidUntil      *string  `json:"validUntil,omitempty"`
	Status          string   `json:"status"`
}

func entitlementToJSON(e repoBilling.Entitlement) entitlementJSON {
	out := entitlementJSON{
		ID:              e.ID.String(),
		EntitlementType: e.EntitlementType,
		AmountPaidCents: e.AmountPaidCents,
		SubtotalCents:   e.SubtotalCents,
		TaxAmountCents:  e.TaxAmountCents,
		TaxType:         e.TaxType,
		TaxJurisdiction: e.TaxJurisdiction,
		ReverseCharge:   e.ReverseCharge,
		Currency:        e.Currency,
		ValidFrom:       e.ValidFrom.UTC().Format(time.RFC3339),
		Status:          e.Status,
	}
	if e.CourseID != nil {
		s := e.CourseID.String()
		out.CourseID = &s
	}
	if e.InvoiceID != nil {
		s := e.InvoiceID.String()
		out.InvoiceID = &s
	}
	if e.ValidUntil != nil {
		s := e.ValidUntil.UTC().Format(time.RFC3339)
		out.ValidUntil = &s
	}
	return out
}

func (d Deps) handleBillingCheckout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
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
		var body struct {
			CourseID      *string `json:"courseId"`
			Plan          string  `json:"plan"`
			PromoCode     string  `json:"promoCode"`
			AffiliateCode string  `json:"affiliateCode"`
			SuccessURL    string  `json:"successUrl"`
			CancelURL     string  `json:"cancelUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		successURL := strings.TrimSpace(body.SuccessURL)
		cancelURL := strings.TrimSpace(body.CancelURL)
		if successURL == "" || cancelURL == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "successUrl and cancelUrl are required.")
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
		result, err := svcBilling.CreateCheckoutSession(r.Context(), d.Pool, cfg, svcBilling.CheckoutRequest{
			UserID:             userID,
			Email:              email,
			CourseID:           courseID,
			Plan:               body.Plan,
			PromoCode:          body.PromoCode,
			AffiliateCode:      body.AffiliateCode,
			SuccessURL:         successURL,
			CancelURL:          cancelURL,
			PlatformTaxEnabled: d.effectiveConfig().FFTaxCollection,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not start checkout.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"sessionId":   result.SessionID,
			"checkoutUrl": result.CheckoutURL,
		})
	}
}

func (d Deps) handleBillingPortal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.billingFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		returnURL := strings.TrimSpace(r.URL.Query().Get("return_url"))
		if returnURL == "" {
			returnURL = strings.TrimRight(d.effectiveConfig().PublicWebOrigin, "/") + "/me/billing"
		}
		email, err := svcBilling.LookupUserEmail(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "User not found.")
			return
		}
		cfg := svcBilling.ConfigFrom(d.effectiveConfig())
		url, err := svcBilling.CreatePortalSession(r.Context(), d.Pool, cfg, userID, email, returnURL)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not open billing portal.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"portalUrl": url})
	}
}

func (d Deps) handleMyEntitlements() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.billingFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		rows, err := repoBilling.ListActiveByUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load entitlements.")
			return
		}
		items := make([]entitlementJSON, 0, len(rows))
		for _, row := range rows {
			items = append(items, entitlementToJSON(row))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entitlements": items})
	}
}

type purchaseJSON struct {
	CourseCode        string  `json:"courseCode"`
	CourseID          string  `json:"courseId"`
	Title             string  `json:"title"`
	PriceCents        int     `json:"priceCents"`
	Currency          string  `json:"currency"`
	Source            string  `json:"source"`
	AcquiredAt        string  `json:"acquiredAt"`
	ReceiptURL        *string `json:"receiptUrl,omitempty"`
	EntitlementID     string  `json:"entitlementId"`
}

func (d Deps) handleMyPurchases() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.courseMarketplaceOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		rows, err := repoBilling.ListMyPurchases(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load purchases.")
			return
		}
		billingURL := strings.TrimRight(d.effectiveConfig().PublicWebOrigin, "/") + "/me/billing"
		items := make([]purchaseJSON, 0, len(rows))
		for _, row := range rows {
			item := purchaseJSON{
				CourseCode:    row.CourseCode,
				CourseID:      row.CourseID.String(),
				Title:         row.Title,
				PriceCents:    row.AmountPaidCents,
				Currency:      row.Currency,
				Source:        row.AcquisitionSource,
				AcquiredAt:    row.AcquiredAt.UTC().Format(time.RFC3339),
				EntitlementID: row.EntitlementID.String(),
			}
			if row.HasReceipt {
				u := billingURL
				item.ReceiptURL = &u
			}
			items = append(items, item)
		}
		telemetry.RecordMyPurchasesView()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"purchases": items})
	}
}

func (d Deps) handleInternalEntitlementCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		callerID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.billingFeatureOff(w) {
			return
		}
		rawUser := strings.TrimSpace(r.URL.Query().Get("user_id"))
		rawCourse := strings.TrimSpace(r.URL.Query().Get("course_id"))
		userID, err := uuid.Parse(rawUser)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "user_id is required.")
			return
		}
		courseID, err := uuid.Parse(rawCourse)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "course_id is required.")
			return
		}
		if userID != callerID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		allowed, err := svcBilling.UserHasCourseAccess(r.Context(), d.Pool, userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not check entitlement.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"entitled": allowed})
	}
}

func (d Deps) handleStripeWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.effectiveConfig().FFStripeBilling && !d.effectiveConfig().FFPaymentsEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		cfg := paymentprovider.ConfigFrom(d.effectiveConfig())
		factory := paymentprovider.Factory{}
		provider, err := factory.Build(paymentprovider.ProviderStripe, cfg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Stripe not configured.")
			return
		}
		event, err := provider.VerifyWebhook(body, r.Header)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid webhook.")
			return
		}
		headers := map[string]string{}
		for k := range r.Header {
			headers[k] = r.Header.Get(k)
		}
		if d.effectiveConfig().FFPaymentsEnabled {
			_, _, err = repoPayments.EnqueueWebhook(r.Context(), d.Pool, repoPayments.ProviderStripe, event.ID, body, headers)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not enqueue webhook.")
				return
			}
			if event.Type == "invoice.payment_failed" {
				d.notifyPaymentFailed(r, body)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		stripeCfg := svcBilling.ConfigFrom(d.effectiveConfig())
		sig := r.Header.Get("Stripe-Signature")
		result, err := svcBilling.HandleWebhook(r.Context(), d.Pool, stripeCfg, body, sig, svcBilling.WebhookOptions{
			RevenueShareEnabled:  d.effectiveConfig().FFRevenueShare,
			TaxCollectionEnabled: d.effectiveConfig().FFTaxCollection,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid webhook.")
			return
		}
		if result != nil && result.EventType == "invoice.payment_failed" {
			d.notifyPaymentFailed(r, body)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) notifyPaymentFailed(r *http.Request, body []byte) {
	if !d.effectiveConfig().EmailNotificationsEnabled || d.Pool == nil {
		return
	}
	var payload struct {
		Data struct {
			Object struct {
				Customer string `json:"customer"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return
	}
	customerID := strings.TrimSpace(payload.Data.Object.Customer)
	if customerID == "" {
		return
	}
	var userID uuid.UUID
	if err := d.Pool.QueryRow(r.Context(), `
SELECT id FROM "user".users WHERE stripe_customer_id = $1
`, customerID).Scan(&userID); err != nil {
		return
	}
	svc := &notifications.Service{Pool: d.Pool, Config: d.effectiveConfig()}
	_ = svc.EnqueueEmail(r.Context(), userID, notificationevents.PaymentFailed, "payment_failed", map[string]string{
		"billingUrl": strings.TrimRight(d.effectiveConfig().PublicWebOrigin, "/") + "/me/billing",
	}, nil)
}
