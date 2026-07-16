package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/transcriptorder"
	"github.com/lextures/lextures/server/internal/repos/organization"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
)

func (d Deps) registerTranscriptLifecycleRoutes(r chi.Router) {
	r.Get("/api/v1/admin/transcripts/orders", d.handleAdminListTranscriptOrders())
	r.Get("/api/v1/admin/transcripts/orders/{id}", d.handleAdminGetTranscriptOrder())
	r.Post("/api/v1/admin/transcripts/orders/{id}/transition", d.handleAdminTransitionTranscriptOrder())
	r.Get("/api/v1/admin/transcripts/holds", d.handleAdminListTranscriptHolds())
	r.Post("/api/v1/admin/transcripts/holds", d.handleAdminPlaceTranscriptHold())
	r.Post("/api/v1/admin/transcripts/holds/{id}/release", d.handleAdminReleaseTranscriptHold())
	r.Post("/api/v1/integrations/transcripts/holds", d.handleIntegrationsTranscriptHoldUpsert())
}

type holdJSON struct {
	ID             string  `json:"id"`
	UserID         string  `json:"userId"`
	OrgID          *string `json:"orgId,omitempty"`
	Type           string  `json:"type"`
	Reason         *string `json:"reason,omitempty"`
	StudentMessage string  `json:"studentMessage"`
	ExternalID     *string `json:"externalId,omitempty"`
	PlacedBy       *string `json:"placedBy,omitempty"`
	PlacedAt       string  `json:"placedAt"`
	ReleasedBy     *string `json:"releasedBy,omitempty"`
	ReleasedAt     *string `json:"releasedAt,omitempty"`
	Active         bool    `json:"active"`
}

func holdToJSON(h transcriptsrepo.Hold) holdJSON {
	out := holdJSON{
		ID:             h.ID.String(),
		UserID:         h.UserID.String(),
		Type:           string(h.Type),
		Reason:         h.Reason,
		StudentMessage: h.StudentMessageSafe(),
		ExternalID:     h.ExternalID,
		PlacedAt:       h.PlacedAt.UTC().Format(time.RFC3339),
		Active:         h.Active(),
	}
	if h.OrgID != nil {
		s := h.OrgID.String()
		out.OrgID = &s
	}
	if h.PlacedBy != nil {
		s := h.PlacedBy.String()
		out.PlacedBy = &s
	}
	if h.ReleasedBy != nil {
		s := h.ReleasedBy.String()
		out.ReleasedBy = &s
	}
	if h.ReleasedAt != nil {
		s := h.ReleasedAt.UTC().Format(time.RFC3339)
		out.ReleasedAt = &s
	}
	return out
}

type adminOrderJSON struct {
	orderJSON
	UserID          string `json:"userId"`
	UserEmail       string `json:"userEmail"`
	ActiveHoldCount int    `json:"activeHoldCount"`
}

func adminOrderToJSON(row transcriptsrepo.AdminOrderRow, events []transcriptsrepo.OrderEvent) adminOrderJSON {
	rej := transcriptsrepo.RejectionReasonFromEvents(events)
	base := orderToJSONExt(&row.Order, nil, events, rej)
	base.OnHold = row.Status == transcriptsrepo.OrderOnHold || row.ActiveHoldCount > 0
	if row.OldestHoldMsg != nil {
		msg := strings.TrimSpace(*row.OldestHoldMsg)
		if msg == "" {
			msg = transcriptorder.DefaultStudentMessage(transcriptorder.HoldOther)
		}
		base.StudentMessage = &msg
	}
	return adminOrderJSON{
		orderJSON:       base,
		UserID:          row.UserID.String(),
		UserEmail:       row.UserEmail,
		ActiveHoldCount: row.ActiveHoldCount,
	}
}

// GET /api/v1/admin/transcripts/orders?status=&hold=&q=
func (d Deps) handleAdminListTranscriptOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		f := transcriptsrepo.AdminOrderListFilter{
			Status: strings.TrimSpace(r.URL.Query().Get("status")),
			Query:  strings.TrimSpace(r.URL.Query().Get("q")),
		}
		if raw := strings.TrimSpace(r.URL.Query().Get("hold")); raw != "" {
			v := raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
			if raw == "0" || strings.EqualFold(raw, "false") || strings.EqualFold(raw, "no") {
				v = false
				f.Hold = &v
			} else if v {
				f.Hold = &v
			}
		}
		if lim := strings.TrimSpace(r.URL.Query().Get("limit")); lim != "" {
			if n, err := strconv.Atoi(lim); err == nil {
				f.Limit = n
			}
		}
		list, err := transcriptsrepo.ListAdminOrders(r.Context(), d.Pool, f)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load fulfillment queue.")
			return
		}
		out := make([]adminOrderJSON, 0, len(list))
		for _, row := range list {
			events, _ := transcriptsrepo.ListOrderEvents(r.Context(), d.Pool, row.ID)
			out = append(out, adminOrderToJSON(row, events))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"orders": out})
	}
}

// GET /api/v1/admin/transcripts/orders/{id}
func (d Deps) handleAdminGetTranscriptOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		order, err := transcriptsrepo.GetOrderByID(r.Context(), d.Pool, orderID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		holds, _ := transcriptsrepo.ListActiveHoldsForUser(r.Context(), d.Pool, order.UserID, order.OrgID)
		events, _ := transcriptsrepo.ListOrderEvents(r.Context(), d.Pool, order.ID)
		rej := transcriptsrepo.RejectionReasonFromEvents(events)
		row := transcriptsrepo.AdminOrderRow{Order: *order, ActiveHoldCount: len(holds)}
		if len(holds) > 0 {
			msg := holds[0].StudentMessageSafe()
			row.OldestHoldMsg = &msg
		}
		// Resolve email for admin view
		_ = d.Pool.QueryRow(r.Context(), `SELECT email FROM "user".users WHERE id = $1`, order.UserID).Scan(&row.UserEmail)
		out := adminOrderToJSON(row, events)
		out.orderJSON = orderToJSONExt(order, holds, events, rej)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"order": out})
	}
}

type transitionBody struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// POST /api/v1/admin/transcripts/orders/{id}/transition
func (d Deps) handleAdminTransitionTranscriptOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = r.Body.Close()
		var body transitionBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		action, err := transcriptorder.ParseAction(body.Action)
		if err != nil || action == transcriptorder.ActionSubmit {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"action must be approve, reject, cancel, complete, hold, or release.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		order, err := transcriptsrepo.TransitionOrder(r.Context(), d.Pool, cfg, transcriptsrepo.TransitionInput{
			OrderID:      orderID,
			ActorID:      &actorID,
			Action:       action,
			Reason:       body.Reason,
			AutoApproval: cfg.AutoApprovalEnabled,
		})
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		holds, _ := transcriptsrepo.ListActiveHoldsForUser(r.Context(), d.Pool, order.UserID, order.OrgID)
		events, _ := transcriptsrepo.ListOrderEvents(r.Context(), d.Pool, order.ID)
		rej := transcriptsrepo.RejectionReasonFromEvents(events)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order": orderToJSONExt(order, holds, events, rej),
		})
	}
}

// GET /api/v1/admin/transcripts/holds?userId=&active=true
func (d Deps) handleAdminListTranscriptHolds() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		var userID *uuid.UUID
		if raw := strings.TrimSpace(r.URL.Query().Get("userId")); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "userId must be a UUID.")
				return
			}
			userID = &id
		}
		activeOnly := true
		if raw := strings.TrimSpace(r.URL.Query().Get("active")); raw != "" {
			activeOnly = raw == "1" || strings.EqualFold(raw, "true")
		}
		list, err := transcriptsrepo.ListHolds(r.Context(), d.Pool, nil, userID, activeOnly, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load holds.")
			return
		}
		out := make([]holdJSON, 0, len(list))
		for _, h := range list {
			out = append(out, holdToJSON(h))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"holds": out})
	}
}

type placeHoldBody struct {
	UserID         string  `json:"userId"`
	Type           string  `json:"type"`
	Reason         *string `json:"reason"`
	StudentMessage *string `json:"studentMessage"`
	ExternalID     *string `json:"externalId"`
}

// POST /api/v1/admin/transcripts/holds
func (d Deps) handleAdminPlaceTranscriptHold() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		_ = r.Body.Close()
		var body placeHoldBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		userID, err := uuid.Parse(strings.TrimSpace(body.UserID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "userId must be a UUID.")
			return
		}
		holdType, err := transcriptorder.ParseHoldType(body.Type)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"type must be financial, disciplinary, registrar, library, or other.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		msg := body.StudentMessage
		if msg == nil || strings.TrimSpace(*msg) == "" {
			def := transcriptorder.DefaultStudentMessage(holdType)
			msg = &def
		}
		hold, err := transcriptsrepo.PlaceHold(r.Context(), d.Pool, transcriptsrepo.PlaceHoldInput{
			UserID:         userID,
			OrgID:          &orgID,
			Type:           holdType,
			Reason:         body.Reason,
			StudentMessage: msg,
			ExternalID:     body.ExternalID,
			PlacedBy:       &actorID,
		})
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		cfg, _ := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		_ = transcriptsrepo.ReevaluateOrdersAfterHoldChange(r.Context(), d.Pool, cfg, userID, &orgID, &actorID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"hold": holdToJSON(*hold)})
	}
}

// POST /api/v1/admin/transcripts/holds/{id}/release
func (d Deps) handleAdminReleaseTranscriptHold() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		holdID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid hold id.")
			return
		}
		hold, err := transcriptsrepo.ReleaseHold(r.Context(), d.Pool, holdID, &actorID)
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		cfg, _ := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		_ = transcriptsrepo.ReevaluateOrdersAfterHoldChange(r.Context(), d.Pool, cfg, hold.UserID, hold.OrgID, &actorID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"hold": holdToJSON(*hold)})
	}
}

type integrationHoldBody struct {
	UserID         string  `json:"userId"`
	UserEmail      string  `json:"userEmail"`
	Type           string  `json:"type"`
	Reason         *string `json:"reason"`
	StudentMessage *string `json:"studentMessage"`
	ExternalID     string  `json:"externalId"`
	Released       bool    `json:"released"`
}

// POST /api/v1/integrations/transcripts/holds — HMAC-authenticated SIS upsert.
func (d Deps) handleIntegrationsTranscriptHoldUpsert() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcripts config.")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read request body.")
			return
		}
		secret := ""
		if cfg.WebhookSecret != nil {
			secret = strings.TrimSpace(*cfg.WebhookSecret)
		}
		if !verifyTranscriptsHoldHMAC(r, secret, body) {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid webhook signature.")
			return
		}
		var payload integrationHoldBody
		if err := json.Unmarshal(body, &payload); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		ext := strings.TrimSpace(payload.ExternalID)
		if ext == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "externalId is required.")
			return
		}
		holdType, err := transcriptorder.ParseHoldType(payload.Type)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "type is invalid.")
			return
		}
		userID, err := resolveIntegrationHoldUser(r, d, payload)
		if err != nil || userID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "userId or userEmail is required.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		if payload.Released {
			existing, findErr := transcriptsrepo.FindHoldByExternalID(r.Context(), d.Pool, &orgID, ext)
			if findErr == nil && existing != nil && existing.Active() {
				hold, relErr := transcriptsrepo.ReleaseHold(r.Context(), d.Pool, existing.ID, nil)
				if relErr != nil {
					writeOrderRepoError(w, relErr)
					return
				}
				_ = transcriptsrepo.ReevaluateOrdersAfterHoldChange(r.Context(), d.Pool, cfg, userID, &orgID, nil)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]any{"hold": holdToJSON(*hold)})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		msg := payload.StudentMessage
		if msg == nil || strings.TrimSpace(*msg) == "" {
			def := transcriptorder.DefaultStudentMessage(holdType)
			msg = &def
		}
		hold, err := transcriptsrepo.UpsertExternalHold(r.Context(), d.Pool, transcriptsrepo.PlaceHoldInput{
			UserID:         userID,
			OrgID:          &orgID,
			Type:           holdType,
			Reason:         payload.Reason,
			StudentMessage: msg,
			ExternalID:     &ext,
		})
		if err != nil {
			writeOrderRepoError(w, err)
			return
		}
		_ = transcriptsrepo.ReevaluateOrdersAfterHoldChange(r.Context(), d.Pool, cfg, userID, &orgID, nil)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"hold": holdToJSON(*hold)})
	}
}

func resolveIntegrationHoldUser(r *http.Request, d Deps, payload integrationHoldBody) (uuid.UUID, error) {
	if raw := strings.TrimSpace(payload.UserID); raw != "" {
		return uuid.Parse(raw)
	}
	email := strings.TrimSpace(strings.ToLower(payload.UserEmail))
	if email == "" {
		return uuid.Nil, errIntegrationUserRequired
	}
	var id uuid.UUID
	err := d.Pool.QueryRow(r.Context(), `
SELECT id FROM "user".users WHERE LOWER(email) = $1 LIMIT 1
`, email).Scan(&id)
	return id, err
}

type errIntegrationUser string

func (e errIntegrationUser) Error() string { return string(e) }

var errIntegrationUserRequired = errIntegrationUser("user required")

func verifyTranscriptsHoldHMAC(r *http.Request, secret string, body []byte) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	sig := strings.TrimSpace(r.Header.Get("X-Lextures-Signature"))
	if strings.HasPrefix(strings.ToLower(sig), "sha256=") {
		sig = sig[7:]
	}
	if sig == "" {
		sig = strings.TrimSpace(r.Header.Get("X-Hub-Signature-256"))
		if strings.HasPrefix(strings.ToLower(sig), "sha256=") {
			sig = sig[7:]
		}
	}
	if sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(strings.ToLower(sig)), []byte(strings.ToLower(expected))) == 1
}
