package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/transcriptorder"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
)

func (d Deps) registerTranscriptTrackingRoutes(r chi.Router) {
	r.Get("/api/v1/transcripts/orders/{id}/timeline", d.handleGetTranscriptOrderTimeline())
	r.Post("/api/v1/transcripts/orders/{id}/cancel", d.handleCancelTranscriptOrder())
}

func timelineEntryJSON(e transcriptsrepo.TimelineEntry) map[string]any {
	out := map[string]any{
		"id":     e.ID,
		"kind":   string(e.Kind),
		"at":     e.At.UTC().Format(time.RFC3339),
		"status": e.Status,
		"label":  e.Label,
	}
	if e.ItemID != nil {
		out["itemId"] = e.ItemID.String()
	}
	if e.Adapter != nil {
		out["adapter"] = *e.Adapter
	}
	if e.AttemptNo != nil {
		out["attemptNo"] = *e.AttemptNo
	}
	if e.Detail != nil && *e.Detail != "" {
		out["detail"] = *e.Detail
	}
	if e.Reason != nil && *e.Reason != "" {
		out["reason"] = *e.Reason
	}
	return out
}

// GET /api/v1/transcripts/orders/{id}/timeline
func (d Deps) handleGetTranscriptOrderTimeline() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		tl, err := transcriptsrepo.BuildOrderTimeline(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		entries := make([]map[string]any, 0, len(tl.Entries))
		for _, e := range tl.Entries {
			entries = append(entries, timelineEntryJSON(e))
		}
		resend := make([]string, 0, len(tl.CanResendItems))
		for _, id := range tl.CanResendItems {
			resend = append(resend, id.String())
		}
		items := make([]orderItemJSON, 0, len(tl.Items))
		for _, it := range tl.Items {
			item := orderItemJSON{
				ID:             it.ID.String(),
				DeliveryMethod: string(it.DeliveryMethod),
				Urgency:        string(it.Urgency),
				Status:         string(it.Status),
				CreatedAt:      it.CreatedAt.UTC().Format(time.RFC3339),
			}
			if it.RecipientID != nil {
				s := it.RecipientID.String()
				item.RecipientID = &s
			}
			if it.DocumentID != nil {
				s := it.DocumentID.String()
				item.DocumentID = &s
			}
			if it.Recipient != nil {
				rj := recipientToJSON(*it.Recipient)
				item.Recipient = &rj
			}
			items = append(items, item)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"timeline": map[string]any{
				"orderId":        tl.OrderID.String(),
				"status":         string(tl.Status),
				"canCancel":      tl.CanCancel,
				"canResendItems": resend,
				"entries":        entries,
				"items":          items,
			},
		})
	}
}

// POST /api/v1/transcripts/orders/{id}/cancel — learner self-service cancel + refund when paid.
func (d Deps) handleCancelTranscriptOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		o, err := transcriptsrepo.GetOrderForUser(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		if !transcriptsrepo.LearnerCancelAllowed(o) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Order cannot be canceled in its current state.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		actor := userID
		order, err := transcriptsrepo.TransitionOrder(r.Context(), d.Pool, cfg, transcriptsrepo.TransitionInput{
			OrderID: orderID,
			ActorID: &actor,
			Action:  transcriptorder.ActionCancel,
			Reason:  "canceled by student",
		})
		if err != nil {
			if errors.Is(err, transcriptsrepo.ErrIllegalOrderTransition) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Order cannot be canceled in its current state.")
				return
			}
			writeOrderRepoError(w, err)
			return
		}

		var refundAny map[string]any
		if order.PaymentStatus == transcriptsrepo.OrderPaymentPaid ||
			order.PaymentStatus == transcriptsrepo.OrderPaymentPartiallyRefunded {
			stripeCfg := svcBilling.ConfigFrom(d.effectiveConfig())
			if stripeCfg.IsConfigured() {
				refund, updated, rerr := svcBilling.RefundTranscriptOrder(r.Context(), d.Pool, stripeCfg, orderID, nil)
				if rerr != nil {
					slog.Warn("transcript.cancel.refund", "err", rerr, "order_id", orderID.String())
				} else {
					order = updated
					if refund != nil {
						refundAny = map[string]any{
							"refundId":    refund.RefundID,
							"amountCents": refund.AmountCents,
							"currency":    refund.Currency,
							"status":      refund.Status,
						}
					}
				}
			}
		}

		out := map[string]any{"order": orderToJSON(order)}
		if refundAny != nil {
			out["refund"] = refundAny
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
