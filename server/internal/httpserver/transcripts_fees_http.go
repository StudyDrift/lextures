package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/transcriptfees"
	"github.com/lextures/lextures/server/internal/repos/organization"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/repos/user"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
)

func (d Deps) registerTranscriptFeeRoutes(r chi.Router) {
	r.Get("/api/v1/admin/transcripts/fees", d.handleGetAdminTranscriptFees())
	r.Put("/api/v1/admin/transcripts/fees", d.handlePutAdminTranscriptFees())
	r.Get("/api/v1/admin/transcripts/waiver-codes", d.handleListAdminWaiverCodes())
	r.Post("/api/v1/admin/transcripts/waiver-codes", d.handlePostAdminWaiverCode())
	r.Post("/api/v1/admin/transcripts/orders/{id}/waive", d.handleAdminWaiveTranscriptOrder())
	r.Post("/api/v1/admin/transcripts/orders/{id}/refund", d.handleAdminRefundTranscriptOrder())

	r.Get("/api/v1/transcripts/orders/{id}/quote", d.handleGetTranscriptOrderQuote())
	r.Post("/api/v1/transcripts/orders/{id}/checkout", d.handlePostTranscriptOrderCheckout())
	r.Get("/api/v1/transcripts/orders/{id}/receipt", d.handleGetTranscriptOrderReceipt())
}

type feeScheduleJSON struct {
	OrgID            string         `json:"orgId"`
	Currency         string         `json:"currency"`
	BaseFee          int            `json:"baseFee"`
	RushFee          int            `json:"rushFee"`
	PerRecipientFee  int            `json:"perRecipientFee"`
	MethodSurcharges map[string]int `json:"methodSurcharges"`
	FreeAllotment    int            `json:"freeAllotment"`
	AllotmentPeriod  string         `json:"allotmentPeriod"`
	UpdatedAt        string         `json:"updatedAt,omitempty"`
}

func feeScheduleToJSON(s *transcriptsrepo.FeeSchedule) feeScheduleJSON {
	out := feeScheduleJSON{
		OrgID:            s.OrgID.String(),
		Currency:         s.Currency,
		BaseFee:          s.BaseFee,
		RushFee:          s.RushFee,
		PerRecipientFee:  s.PerRecipientFee,
		MethodSurcharges: s.MethodSurcharges,
		FreeAllotment:    s.FreeAllotment,
		AllotmentPeriod:  s.AllotmentPeriod,
	}
	if out.MethodSurcharges == nil {
		out.MethodSurcharges = map[string]int{}
	}
	if !s.UpdatedAt.IsZero() {
		out.UpdatedAt = s.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

type waiverCodeJSON struct {
	ID        string  `json:"id"`
	OrgID     string  `json:"orgId"`
	Code      string  `json:"code"`
	Kind      string  `json:"kind"`
	Value     *int    `json:"value,omitempty"`
	MaxUses   *int    `json:"maxUses,omitempty"`
	UsedCount int     `json:"usedCount"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

func waiverCodeToJSON(c transcriptsrepo.WaiverCode) waiverCodeJSON {
	out := waiverCodeJSON{
		ID:        c.ID.String(),
		OrgID:     c.OrgID.String(),
		Code:      c.Code,
		Kind:      c.Kind,
		Value:     c.Value,
		MaxUses:   c.MaxUses,
		UsedCount: c.UsedCount,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
	}
	if c.ExpiresAt != nil {
		s := c.ExpiresAt.UTC().Format(time.RFC3339)
		out.ExpiresAt = &s
	}
	return out
}

// GET /api/v1/admin/transcripts/fees
func (d Deps) handleGetAdminTranscriptFees() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		sched, err := transcriptsrepo.GetFeeSchedule(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load fee schedule.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(feeScheduleToJSON(sched))
	}
}

type putFeesBody struct {
	Currency         string         `json:"currency"`
	BaseFee          int            `json:"baseFee"`
	RushFee          int            `json:"rushFee"`
	PerRecipientFee  int            `json:"perRecipientFee"`
	MethodSurcharges map[string]int `json:"methodSurcharges"`
	FreeAllotment    int            `json:"freeAllotment"`
	AllotmentPeriod  string         `json:"allotmentPeriod"`
}

// PUT /api/v1/admin/transcripts/fees
func (d Deps) handlePutAdminTranscriptFees() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body putFeesBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		sched, err := transcriptsrepo.UpsertFeeSchedule(r.Context(), d.Pool, transcriptsrepo.UpsertFeeScheduleInput{
			OrgID:            orgID,
			Currency:         body.Currency,
			BaseFee:          body.BaseFee,
			RushFee:          body.RushFee,
			PerRecipientFee:  body.PerRecipientFee,
			MethodSurcharges: body.MethodSurcharges,
			FreeAllotment:    body.FreeAllotment,
			AllotmentPeriod:  body.AllotmentPeriod,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(feeScheduleToJSON(sched))
	}
}

// GET /api/v1/admin/transcripts/waiver-codes
func (d Deps) handleListAdminWaiverCodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		list, err := transcriptsrepo.ListWaiverCodes(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list waiver codes.")
			return
		}
		out := make([]waiverCodeJSON, 0, len(list))
		for _, c := range list {
			out = append(out, waiverCodeToJSON(c))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"waiverCodes": out})
	}
}

type postWaiverCodeBody struct {
	Code      string  `json:"code"`
	Kind      string  `json:"kind"`
	Value     *int    `json:"value"`
	MaxUses   *int    `json:"maxUses"`
	ExpiresAt *string `json:"expiresAt"`
}

// POST /api/v1/admin/transcripts/waiver-codes
func (d Deps) handlePostAdminWaiverCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postWaiverCodeBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var expires *time.Time
		if body.ExpiresAt != nil && strings.TrimSpace(*body.ExpiresAt) != "" {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.ExpiresAt))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "expiresAt must be RFC3339.")
				return
			}
			expires = &t
		}
		code, err := transcriptsrepo.CreateWaiverCode(r.Context(), d.Pool, transcriptsrepo.CreateWaiverCodeInput{
			OrgID:     orgID,
			Code:      body.Code,
			Kind:      body.Kind,
			Value:     body.Value,
			MaxUses:   body.MaxUses,
			ExpiresAt: expires,
			CreatedBy: &userID,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(waiverCodeToJSON(*code))
	}
}

type waiveBody struct {
	Reason string `json:"reason"`
}

// POST /api/v1/admin/transcripts/orders/{id}/waive
func (d Deps) handleAdminWaiveTranscriptOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body waiveBody
		_ = json.Unmarshal(b, &body)
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		order, err := transcriptsrepo.AdminWaiveOrder(r.Context(), d.Pool, cfg, orderID, userID, body.Reason)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"order": orderToJSON(order)})
	}
}

type refundBody struct {
	AmountCents *int `json:"amountCents"`
}

// POST /api/v1/admin/transcripts/orders/{id}/refund
func (d Deps) handleAdminRefundTranscriptOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body refundBody
		_ = json.Unmarshal(b, &body)
		stripeCfg := svcBilling.ConfigFrom(d.effectiveConfig())
		refund, order, err := svcBilling.RefundTranscriptOrder(r.Context(), d.Pool, stripeCfg, orderID, body.AmountCents)
		if err != nil {
			if errors.Is(err, transcriptsrepo.ErrRefundNotAllowed) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Order cannot be refunded.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order": orderToJSON(order),
			"refund": map[string]any{
				"refundId":    refund.RefundID,
				"amountCents": refund.AmountCents,
				"currency":    refund.Currency,
				"status":      refund.Status,
			},
		})
	}
}

// GET /api/v1/transcripts/orders/{id}/quote
func (d Deps) handleGetTranscriptOrderQuote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		o, err := transcriptsrepo.GetOrderForUser(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		waiverCode := strings.TrimSpace(r.URL.Query().Get("waiverCode"))
		q, _, err := transcriptsrepo.QuoteOrder(r.Context(), d.Pool, cfg, o, transcriptsrepo.QuoteOptions{
			WaiverCode: waiverCode,
		})
		if err != nil {
			if errors.Is(err, transcriptsrepo.ErrWaiverCodeNotFound) ||
				errors.Is(err, transcriptsrepo.ErrWaiverCodeInvalid) ||
				errors.Is(err, transcriptsrepo.ErrWaiverCodeExhausted) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not compute quote.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"orderId":       o.ID.String(),
			"feesEnabled":   cfg.FeesEnabled,
			"paymentStatus": string(o.PaymentStatus),
			"quote":         q,
		})
	}
}

type checkoutBody struct {
	WaiverCode string `json:"waiverCode"`
	SuccessURL string `json:"successUrl"`
	CancelURL  string `json:"cancelUrl"`
}

// POST /api/v1/transcripts/orders/{id}/checkout
func (d Deps) handlePostTranscriptOrderCheckout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body checkoutBody
		_ = json.Unmarshal(b, &body)

		// Optional: apply waiver code before checkout.
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		o, err := transcriptsrepo.GetOrderForUser(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		if code := strings.TrimSpace(body.WaiverCode); code != "" {
			if _, err := transcriptsrepo.ApplyWaiverCodeToOrder(r.Context(), d.Pool, cfg, o, code, &userID); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			o, err = transcriptsrepo.GetOrderForUser(r.Context(), d.Pool, orderID, userID)
			if err != nil {
				writeOrderRepoError(w, err)
				return
			}
			if ok, _ := transcriptsrepo.PaymentSatisfiedForOrder(r.Context(), d.Pool, cfg, o); ok {
				advanced, _ := transcriptsrepo.AdvanceAfterPayment(r.Context(), d.Pool, cfg, orderID, &userID)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"paid":  false,
					"waived": true,
					"order": orderToJSON(advanced),
				})
				return
			}
		}

		u, err := user.FindByID(r.Context(), d.Pool, userID)
		if err != nil || u == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load user.")
			return
		}
		stripeCfg := svcBilling.ConfigFrom(d.effectiveConfig())
		result, err := svcBilling.StartTranscriptCheckout(r.Context(), d.Pool, stripeCfg, cfg, svcBilling.TranscriptCheckoutRequest{
			UserID:     userID,
			Email:      u.Email,
			OrderID:    orderID,
			SuccessURL: body.SuccessURL,
			CancelURL:  body.CancelURL,
			WaiverCode: body.WaiverCode,
		})
		if err != nil {
			if errors.Is(err, transcriptsrepo.ErrPaymentNotRequired) {
				advanced, _ := transcriptsrepo.AdvanceAfterPayment(r.Context(), d.Pool, cfg, orderID, &userID)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"paid":   false,
					"waived": true,
					"order":  orderToJSON(advanced),
				})
				return
			}
			if errors.Is(err, transcriptsrepo.ErrPaymentAlreadyDone) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Payment already satisfied.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sessionId":   result.SessionID,
			"checkoutUrl": result.CheckoutURL,
		})
	}
}

// GET /api/v1/transcripts/orders/{id}/receipt
func (d Deps) handleGetTranscriptOrderReceipt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		o, err := transcriptsrepo.GetOrderForUser(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		switch o.PaymentStatus {
		case transcriptsrepo.OrderPaymentPaid, transcriptsrepo.OrderPaymentWaived, transcriptsrepo.OrderPaymentFree,
			transcriptsrepo.OrderPaymentRefunded, transcriptsrepo.OrderPaymentPartiallyRefunded:
		default:
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Receipt is not available until payment is settled.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		q, _, _ := transcriptsrepo.QuoteOrder(r.Context(), d.Pool, cfg, o, transcriptsrepo.QuoteOptions{SkipFreeAllotment: true})
		amount := 0
		cur := "usd"
		if o.TotalAmount != nil {
			amount = *o.TotalAmount
		}
		if o.Currency != nil && *o.Currency != "" {
			cur = *o.Currency
		} else if q != nil {
			cur = q.Currency
			if amount == 0 {
				amount = q.Total
			}
		}
		email := ""
		if u, err := user.FindByID(r.Context(), d.Pool, userID); err == nil && u != nil {
			email = u.Email
		}
		ref := ""
		if o.PaymentRef != nil {
			ref = *o.PaymentRef
		}
		var quoteLines []transcriptfees.QuoteLine
		if q != nil {
			quoteLines = q.Lines
		}
		in := svcBilling.TranscriptReceiptInput{
			OrderID:        o.ID.String(),
			IssuedAt:       time.Now().UTC(),
			StudentEmail:   email,
			Currency:       cur,
			PaymentStatus:  string(o.PaymentStatus),
			PaymentRef:     ref,
			AmountPaid:     amount,
			AmountRefunded: o.AmountRefunded,
			Lines:          quoteLines,
			IsRefund:       o.PaymentStatus == transcriptsrepo.OrderPaymentRefunded || o.PaymentStatus == transcriptsrepo.OrderPaymentPartiallyRefunded,
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "pdf" {
			pdf, err := svcBilling.BuildTranscriptReceiptPDF(in)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not build receipt PDF.")
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", "attachment; filename=\"transcript-receipt-"+o.ID.String()+".pdf\"")
			w.Header().Set("Content-Length", strconv.Itoa(len(pdf)))
			_, _ = w.Write(pdf)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(svcBilling.BuildTranscriptReceiptJSON(in))
	}
}
