package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoPayments "github.com/lextures/lextures/server/internal/repos/payments"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
	"github.com/lextures/lextures/server/internal/service/paymentprovider"
)

func (d Deps) paymentsFeatureOff(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFStripeBilling && !cfg.FFPaymentsEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Payments are not enabled.")
		return true
	}
	return false
}

func (d Deps) registerPaymentsRoutes(r chi.Router) {
	r.Post("/api/v1/checkout", d.handleCheckout())
	r.Get("/api/v1/me/transactions", d.handleMyTransactions())
	r.Post("/webhooks/paypal", d.handlePayPalWebhook())
	r.Post("/api/v1/admin/transactions/{id}/refund", d.handleAdminRefundTransaction())
}

type transactionJSON struct {
	ID             string  `json:"id"`
	CourseID       *string `json:"courseId,omitempty"`
	Provider       string  `json:"provider"`
	ProviderTxnID  string  `json:"providerTxnId"`
	AmountCents    int     `json:"amountCents"`
	Currency       string  `json:"currency"`
	Status         string  `json:"status"`
	SubscriptionID *string `json:"subscriptionId,omitempty"`
	CreatedAt      string  `json:"createdAt"`
}

func transactionToJSON(tx repoPayments.Transaction) transactionJSON {
	out := transactionJSON{
		ID:            tx.ID.String(),
		Provider:      tx.Provider,
		ProviderTxnID: tx.ProviderTxnID,
		AmountCents:   tx.AmountCents,
		Currency:      tx.Currency,
		Status:        tx.Status,
		SubscriptionID: tx.SubscriptionID,
		CreatedAt:     tx.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if tx.CourseID != nil {
		s := tx.CourseID.String()
		out.CourseID = &s
	}
	return out
}

func (d Deps) handleCheckout() http.HandlerFunc {
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
		if d.paymentsFeatureOff(w) {
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
			Provider      string  `json:"provider"`
			Country       string  `json:"country"`
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
		cfg := paymentprovider.ConfigFrom(d.effectiveConfig())
		if !cfg.StripeConfigured() && !cfg.PayPalConfigured() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "No payment provider configured.")
			return
		}
		result, err := paymentprovider.StartCheckout(r.Context(), d.Pool, cfg, paymentprovider.StartCheckoutRequest{
			UserID:             userID,
			Email:              email,
			CourseID:           courseID,
			Plan:               body.Plan,
			Provider:           paymentprovider.ProviderName(strings.TrimSpace(body.Provider)),
			Country:            strings.TrimSpace(body.Country),
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
			"provider":    string(result.Provider),
		})
	}
}

func (d Deps) handleMyTransactions() http.HandlerFunc {
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
		if d.paymentsFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		rows, err := repoPayments.ListByUser(r.Context(), d.Pool, userID, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load transactions.")
			return
		}
		items := make([]transactionJSON, 0, len(rows))
		for _, row := range rows {
			items = append(items, transactionToJSON(row))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"transactions": items})
	}
}

func (d Deps) handlePayPalWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.effectiveConfig().FFPaymentsEnabled {
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
		provider, err := factory.Build(paymentprovider.ProviderPayPal, cfg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "PayPal not configured.")
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
		if _, _, err := repoPayments.EnqueueWebhook(r.Context(), d.Pool, repoPayments.ProviderPayPal, event.ID, body, headers); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not enqueue webhook.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleAdminRefundTransaction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.paymentsFeatureOff(w) {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, actorID, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		txID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid transaction id.")
			return
		}
		var body struct {
			AmountCents *int `json:"amountCents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		tx, err := repoPayments.GetByID(r.Context(), d.Pool, txID)
		if err != nil || tx == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Transaction not found.")
			return
		}
		if tx.Status == repoPayments.StatusRefunded {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Transaction already refunded.")
			return
		}
		cfg := paymentprovider.ConfigFrom(d.effectiveConfig())
		refund, err := paymentprovider.IssueProviderRefund(r.Context(), d.Pool, cfg, tx, body.AmountCents)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not issue refund.")
			return
		}
		targetType := "payment_transaction"
		_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			EventType:  "payment_refund",
			ActorID:    actorID,
			TargetType: &targetType,
			TargetID:   &txID,
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"refundId":    refund.RefundID,
			"amountCents": refund.AmountCents,
			"status":      refund.Status,
		})
	}
}
