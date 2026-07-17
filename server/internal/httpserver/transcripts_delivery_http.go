package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/background"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/transcriptdelivery"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	transcriptLinkResolveLimit = 60
	transcriptLinkWindow       = time.Minute
)

type transcriptLinkAttempt struct {
	count int
	start time.Time
}

var (
	transcriptLinkMu       sync.Mutex
	transcriptLinkAttempts = map[string]*transcriptLinkAttempt{}
)

func transcriptLinkRateLimited(key string, limit int) bool {
	transcriptLinkMu.Lock()
	defer transcriptLinkMu.Unlock()
	now := time.Now()
	e := transcriptLinkAttempts[key]
	if e == nil || now.Sub(e.start) > transcriptLinkWindow {
		transcriptLinkAttempts[key] = &transcriptLinkAttempt{count: 1, start: now}
		return false
	}
	e.count++
	return e.count > limit
}

func (d Deps) registerTranscriptDeliveryRoutes(r chi.Router) {
	r.Get("/api/v1/r/t/{token}", d.handleGetTranscriptShareLink())
	r.Get("/api/v1/r/t/{token}/download", d.handleDownloadTranscriptShareLink())
	r.Get("/api/v1/transcripts/orders/{id}/items/{itemId}/receipts", d.handleListTranscriptItemReceipts())
	r.Post("/api/v1/admin/transcripts/orders/{id}/items/{itemId}/resend", d.handleAdminResendTranscriptItem())
	r.Post("/api/v1/transcripts/orders/{id}/items/{itemId}/resend", d.handleStudentResendTranscriptItem())
	r.Get("/api/v1/admin/transcripts/delivery-config", d.handleGetAdminTranscriptDeliveryConfig())
	r.Put("/api/v1/admin/transcripts/delivery-config", d.handlePutAdminTranscriptDeliveryConfig())
}

type deliveryAttemptJSON struct {
	ID             string  `json:"id"`
	OrderItemID    string  `json:"orderItemId"`
	Adapter        string  `json:"adapter"`
	AttemptNo      int     `json:"attemptNo"`
	Status         string  `json:"status"`
	ResponseCode   *int    `json:"responseCode,omitempty"`
	Detail         *string `json:"detail,omitempty"`
	IdempotencyKey string  `json:"idempotencyKey"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

func deliveryAttemptToJSON(a transcriptsrepo.DeliveryAttempt) deliveryAttemptJSON {
	return deliveryAttemptJSON{
		ID:             a.ID.String(),
		OrderItemID:    a.OrderItemID.String(),
		Adapter:        string(a.Adapter),
		AttemptNo:      a.AttemptNo,
		Status:         string(a.Status),
		ResponseCode:   a.ResponseCode,
		Detail:         a.Detail,
		IdempotencyKey: a.IdempotencyKey,
		CreatedAt:      a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type shareLinkMetaJSON struct {
	Token              string  `json:"token"`
	ExpiresAt          string  `json:"expiresAt"`
	MaxDownloads       int     `json:"maxDownloads"`
	DownloadsRemaining int     `json:"downloadsRemaining"`
	OpenedAt           *string `json:"openedAt,omitempty"`
	DocumentID         string  `json:"documentId"`
	OrderItemID        string  `json:"orderItemId"`
	Expired            bool    `json:"expired"`
	Exhausted          bool    `json:"exhausted"`
	VerifyToken        *string `json:"verifyToken,omitempty"`
	VerificationURL    *string `json:"verificationUrl,omitempty"`
}

// GET /api/v1/r/t/{token}
func (d Deps) handleGetTranscriptShareLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if transcriptLinkRateLimited("meta:"+r.RemoteAddr, transcriptLinkResolveLimit) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many requests. Please try again later.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		token := strings.TrimSpace(chi.URLParam(r, "token"))
		link, err := transcriptsrepo.GetShareLinkByToken(r.Context(), d.Pool, token)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Link not found.")
			return
		}
		now := time.Now().UTC()
		expired := !link.ExpiresAt.After(now)
		exhausted := link.DownloadCount >= link.MaxDownloads
		if !expired && !exhausted {
			ip := clientIP(r)
			link, _ = transcriptsrepo.RecordShareLinkOpen(r.Context(), d.Pool, link.ID, ip)
			_ = transcriptsrepo.RecordOpenedReceipt(r.Context(), d.Pool, link.OrderItemID, transcriptsrepo.DeliverySecureLink, "share link opened")
			telemetry.RecordBusinessEvent("transcript.item.opened")
			if dc, derr := transcriptsrepo.LoadDeliveryItemContext(r.Context(), d.Pool, link.OrderItemID); derr == nil {
				transcriptsrepo.NotifyDeliveryReceipt(r.Context(), d.Pool, &dc.Order, link.OrderItemID, transcriptsrepo.AttemptOpened)
			}
		}
		remaining := link.MaxDownloads - link.DownloadCount
		if remaining < 0 {
			remaining = 0
		}
		out := shareLinkMetaJSON{
			Token:              link.Token,
			ExpiresAt:          link.ExpiresAt.UTC().Format(time.RFC3339),
			MaxDownloads:       link.MaxDownloads,
			DownloadsRemaining: remaining,
			DocumentID:         link.DocumentID.String(),
			OrderItemID:        link.OrderItemID.String(),
			Expired:            expired,
			Exhausted:          exhausted,
		}
		if link.OpenedAt != nil {
			s := link.OpenedAt.UTC().Format(time.RFC3339)
			out.OpenedAt = &s
		}
		if doc, err := transcriptsrepo.GetDocumentByIDAdmin(r.Context(), d.Pool, link.DocumentID); err == nil && doc != nil && doc.VerifyToken != nil {
			tok := *doc.VerifyToken
			out.VerifyToken = &tok
			base := strings.TrimRight(strings.TrimSpace(d.effectiveConfig().PublicWebOrigin), "/")
			if base != "" {
				url := base + "/verify/" + tok
				out.VerificationURL = &url
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// GET /api/v1/r/t/{token}/download
func (d Deps) handleDownloadTranscriptShareLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if transcriptLinkRateLimited("dl:"+r.RemoteAddr, transcriptLinkResolveLimit) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many requests. Please try again later.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		token := strings.TrimSpace(chi.URLParam(r, "token"))
		link, err := transcriptsrepo.ConsumeShareLinkDownload(r.Context(), d.Pool, token, time.Now().UTC(), clientIP(r))
		if errors.Is(err, transcriptsrepo.ErrShareLinkExpired) {
			apierr.WriteJSON(w, http.StatusGone, apierr.CodeNotFound, "This download link has expired.")
			return
		}
		if errors.Is(err, transcriptsrepo.ErrShareLinkExhausted) {
			apierr.WriteJSON(w, http.StatusGone, apierr.CodeNotFound, "Download limit reached for this link.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Link not found.")
			return
		}
		doc, err := transcriptsrepo.GetDocumentByIDAdmin(r.Context(), d.Pool, link.DocumentID)
		if err != nil || doc == nil || len(doc.PDFBytes) == 0 {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Document unavailable.")
			return
		}
		if !transcriptsrepo.VerifyDocumentHash(doc) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Document integrity check failed.")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="transcript.pdf"`)
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(doc.PDFBytes)
	}
}

// GET /api/v1/transcripts/orders/{id}/items/{itemId}/receipts
func (d Deps) handleListTranscriptItemReceipts() http.HandlerFunc {
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
		itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		o, err := transcriptsrepo.GetOrderForUser(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Order not found.")
			return
		}
		var found bool
		for _, it := range o.Items {
			if it.ID == itemID {
				found = true
				break
			}
		}
		if !found {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Order item not found.")
			return
		}
		attempts, err := transcriptsrepo.ListDeliveryAttemptsForItem(r.Context(), d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load receipts.")
			return
		}
		out := make([]deliveryAttemptJSON, 0, len(attempts))
		for _, a := range attempts {
			out = append(out, deliveryAttemptToJSON(a))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"receipts": out})
	}
}

func (d Deps) resendTranscriptItem(w http.ResponseWriter, r *http.Request, orderID, itemID, actorID uuid.UUID, admin bool) {
	o, err := transcriptsrepo.GetOrderByID(r.Context(), d.Pool, orderID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Order not found.")
		return
	}
	if !admin && o.UserID != actorID {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Order not found.")
		return
	}
	var found bool
	for _, it := range o.Items {
		if it.ID == itemID {
			found = true
			break
		}
	}
	if !found {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Order item not found.")
		return
	}
	if err := transcriptdelivery.PrepareResend(r.Context(), d.Pool, itemID); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
		return
	}
	n, _ := transcriptsrepo.NextAttemptNo(r.Context(), d.Pool, itemID)
	jobID, err := background.EnqueueTranscriptDelivery(r.Context(), d.Pool, itemID)
	if err != nil {
		// Unique-key conflict — a job is already in flight.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"orderItemId": itemID.String(),
			"attemptHint": n,
		})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":          true,
		"jobId":       jobID.String(),
		"orderItemId": itemID.String(),
	})
}

// POST /api/v1/admin/transcripts/orders/{id}/items/{itemId}/resend
func (d Deps) handleAdminResendTranscriptItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		d.resendTranscriptItem(w, r, orderID, itemID, userID, true)
	}
}

// POST /api/v1/transcripts/orders/{id}/items/{itemId}/resend
func (d Deps) handleStudentResendTranscriptItem() http.HandlerFunc {
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
		itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		d.resendTranscriptItem(w, r, orderID, itemID, userID, false)
	}
}

type deliveryConfigJSON struct {
	DeliveryV2       bool    `json:"deliveryV2"`
	WebhookURL       string  `json:"webhookUrl"`
	HasWebhookSecret bool    `json:"hasWebhookSecret"`
	WebhookSecret    string  `json:"webhookSecret,omitempty"`
	Adapters         []string `json:"adapters"`
}

// GET /api/v1/admin/transcripts/delivery-config
func (d Deps) handleGetAdminTranscriptDeliveryConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load config.")
			return
		}
		out := deliveryConfigJSON{
			DeliveryV2: cfg.DeliveryV2,
			Adapters: []string{
				string(transcriptsrepo.DeliveryAPIPeer),
				string(transcriptsrepo.DeliverySecureLink),
				string(transcriptsrepo.DeliveryElectronicPDF),
				string(transcriptsrepo.DeliveryElectronicPESC),
				string(transcriptsrepo.DeliveryEDISPEEDE),
				string(transcriptsrepo.DeliveryPostalMail),
			},
		}
		if cfg.WebhookURL != nil {
			out.WebhookURL = *cfg.WebhookURL
		}
		if cfg.WebhookSecret != nil && strings.TrimSpace(*cfg.WebhookSecret) != "" {
			out.HasWebhookSecret = true
			out.WebhookSecret = placeholderSecretResponse
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// PUT /api/v1/admin/transcripts/delivery-config
func (d Deps) handlePutAdminTranscriptDeliveryConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		var body struct {
			DeliveryV2  *bool  `json:"deliveryV2"`
			WebhookURL  string `json:"webhookUrl"`
			WebhookSecret *string `json:"webhookSecret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load config.")
			return
		}
		url := strings.TrimSpace(body.WebhookURL)
		if url == "" && cfg.WebhookURL != nil {
			url = *cfg.WebhookURL
		}
		if url == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "webhookUrl is required.")
			return
		}
		updated, err := transcriptsrepo.UpsertConfig(r.Context(), d.Pool, transcriptsrepo.UpsertConfigInput{
			WebhookURL:    url,
			WebhookSecret: body.WebhookSecret,
			DeliveryV2:    body.DeliveryV2,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save config.")
			return
		}
		out := deliveryConfigJSON{
			DeliveryV2: updated.DeliveryV2,
			Adapters: []string{
				string(transcriptsrepo.DeliveryAPIPeer),
				string(transcriptsrepo.DeliverySecureLink),
				string(transcriptsrepo.DeliveryElectronicPDF),
				string(transcriptsrepo.DeliveryElectronicPESC),
				string(transcriptsrepo.DeliveryEDISPEEDE),
				string(transcriptsrepo.DeliveryPostalMail),
			},
		}
		if updated.WebhookURL != nil {
			out.WebhookURL = *updated.WebhookURL
		}
		if updated.WebhookSecret != nil && strings.TrimSpace(*updated.WebhookSecret) != "" {
			out.HasWebhookSecret = true
			out.WebhookSecret = placeholderSecretResponse
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
