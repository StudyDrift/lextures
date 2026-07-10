package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	pfmodel "github.com/lextures/lextures/server/internal/models/productfeedback"
	"github.com/lextures/lextures/server/internal/ratelimit"
	"github.com/lextures/lextures/server/internal/repos/organization"
	pfrepo "github.com/lextures/lextures/server/internal/repos/productfeedback"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	feedbackRateLimitCount  = 10
	feedbackRateLimitWindow = 10 * time.Minute
)

func (d Deps) registerFeedbackRoutes(r chi.Router) {
	r.Post("/api/v1/feedback", d.handlePostFeedback())
}

func (d Deps) registerFeedbackAdminRoutes(r chi.Router) {
	r.Get("/api/v1/admin/feedback", d.handleAdminFeedbackList())
	r.Get("/api/v1/admin/feedback/{id}", d.handleAdminFeedbackGet())
	r.Patch("/api/v1/admin/feedback/{id}", d.handleAdminFeedbackPatch())
}

func (d Deps) feedbackOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFFeedback {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Feedback is not enabled.")
		return true
	}
	return false
}

type submitFeedbackBody struct {
	Message        string             `json:"message"`
	Category       string             `json:"category"`
	Source         string             `json:"source"`
	AppVersion     *string            `json:"app_version"`
	Context        *pfmodel.Context   `json:"context"`
	IdempotencyKey *string            `json:"idempotency_key"`
}

type submitFeedbackResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

func (d Deps) handlePostFeedback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.feedbackOff(w) {
			return
		}
		userID, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		if d.feedbackRateLimited(w, r, userID) {
			return
		}

		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body submitFeedbackBody
		if err := json.Unmarshal(b, &body); err != nil {
			telemetry.RecordFeedbackSubmitError()
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		message, err := pfmodel.ValidateMessage(body.Message)
		if err != nil {
			telemetry.RecordFeedbackSubmitError()
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		declared, err := pfmodel.ParseSource(body.Source)
		if err != nil {
			telemetry.RecordFeedbackSubmitError()
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid source.")
			return
		}
		source := pfmodel.ReconcileSource(declared, r.UserAgent())
		category := pfmodel.NormalizeCategory(body.Category)
		ctxMeta := pfmodel.Context{}
		if body.Context != nil {
			ctxMeta = *body.Context
		}
		ctxMeta.UserAgent = r.UserAgent()

		var orgID *uuid.UUID
		if d.JWTSigner != nil {
			if u, err := auth.UserFromRequest(r, d.JWTSigner); err == nil && u.OrgID != "" {
				if oid, parseErr := uuid.Parse(u.OrgID); parseErr == nil {
					orgID = &oid
				}
			}
		}

		sub, err := pfrepo.Insert(r.Context(), d.Pool, pfrepo.InsertInput{
			UserID:         userID,
			OrgID:          orgID,
			Message:        message,
			Category:       category,
			Source:         source,
			AppVersion:     body.AppVersion,
			Context:        ctxMeta,
			IdempotencyKey: body.IdempotencyKey,
		})
		if err != nil {
			telemetry.RecordFeedbackSubmitError()
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save feedback.")
			return
		}
		telemetry.RecordFeedbackSubmitted(string(source), string(category))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(submitFeedbackResponse{
			ID:        sub.ID.String(),
			CreatedAt: sub.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}

func (d Deps) feedbackRateLimited(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	limiter := d.buildRateLimiter()
	rule := config.RateLimitRule{Limit: feedbackRateLimitCount, Window: feedbackRateLimitWindow}
	key := limiter.UserKey(userID.String(), "feedback")
	dec := limiter.Allow(r.Context(), key, rule, ratelimit.LimitTypeToken)
	if dec.Allowed {
		since := time.Now().UTC().Add(-feedbackRateLimitWindow)
		n, err := pfrepo.CountRecentByUser(r.Context(), d.Pool, userID, since)
		if err == nil && n >= feedbackRateLimitCount {
			dec.Allowed = false
			dec.RetryAfter = int(feedbackRateLimitWindow.Seconds())
		}
	}
	if dec.Allowed {
		return false
	}
	telemetry.RecordFeedbackSubmitError()
	ratelimit.RecordExceeded("feedback", ratelimit.LimitTypeToken)
	w.Header().Set("Retry-After", strconv.Itoa(dec.RetryAfter))
	apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many feedback submissions. Try again later.")
	return true
}

type feedbackListResponse struct {
	Items      []feedbackListItemJSON `json:"items"`
	NextCursor string                 `json:"next_cursor,omitempty"`
	Total      int                    `json:"total,omitempty"`
}

type feedbackListItemJSON struct {
	ID             string              `json:"id"`
	MessagePreview string              `json:"message_preview"`
	Category       string              `json:"category"`
	Source         string              `json:"source"`
	Status         string              `json:"status"`
	Submitter      feedbackPersonJSON  `json:"submitter"`
	CreatedAt      string              `json:"created_at"`
}

type feedbackPersonJSON struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (d Deps) handleAdminFeedbackList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		adminID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		start := time.Now()
		f, valid := d.parseFeedbackListFilter(w, r)
		if !valid {
			return
		}
		items, total, nextCursor, err := pfrepo.List(r.Context(), d.Pool, f)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list feedback.")
			return
		}
		telemetry.ObserveFeedbackAdminList(time.Since(start).Seconds())
		d.recordFeedbackAdminAudit(r, adminID, auditservice.EventFeedbackAdminRead, nil, map[string]any{
			"action": "list",
			"filter": f,
			"count":  len(items),
		})
		out := make([]feedbackListItemJSON, len(items))
		for i, item := range items {
			out[i] = feedbackListItemJSON{
				ID:             item.ID.String(),
				MessagePreview: item.MessagePreview,
				Category:       string(item.Category),
				Source:         string(item.Source),
				Status:         string(item.Status),
				Submitter: feedbackPersonJSON{
					Name:  item.Submitter.Name,
					Email: item.Submitter.Email,
				},
				CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(feedbackListResponse{
			Items:      out,
			NextCursor: nextCursor,
			Total:      total,
		})
	}
}

func (d Deps) parseFeedbackListFilter(w http.ResponseWriter, r *http.Request) (pfrepo.ListFilter, bool) {
	qp := r.URL.Query()
	f := pfrepo.ListFilter{
		Status:   strings.TrimSpace(qp.Get("status")),
		Category: strings.TrimSpace(qp.Get("category")),
		Source:   strings.TrimSpace(qp.Get("source")),
		Query:    strings.TrimSpace(qp.Get("q")),
		Cursor:   strings.TrimSpace(qp.Get("cursor")),
	}
	if lim := strings.TrimSpace(qp.Get("limit")); lim != "" {
		n, err := strconv.Atoi(lim)
		if err != nil || n <= 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid limit.")
			return f, false
		}
		f.Limit = n
	}
	if from := strings.TrimSpace(qp.Get("from")); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid from date.")
			return f, false
		}
		f.From = &t
	}
	if to := strings.TrimSpace(qp.Get("to")); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid to date.")
			return f, false
		}
		f.To = &t
	}
	if f.Status != "" {
		if _, err := pfmodel.ParseStatus(f.Status); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status filter.")
			return f, false
		}
	}
	if f.Category != "" {
		if _, err := pfmodel.ParseCategory(f.Category); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid category filter.")
			return f, false
		}
	}
	if f.Source != "" {
		if _, err := pfmodel.ParseSource(f.Source); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid source filter.")
			return f, false
		}
	}
	if _, err := pfrepo.DecodeCursor(f.Cursor); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid cursor.")
		return f, false
	}
	return f, true
}

type feedbackDetailJSON struct {
	ID          string             `json:"id"`
	UserID      *string            `json:"user_id,omitempty"`
	OrgID       *string            `json:"org_id,omitempty"`
	Message     string             `json:"message"`
	Category    string             `json:"category"`
	Source      string             `json:"source"`
	AppVersion  *string            `json:"app_version,omitempty"`
	Context     pfmodel.Context    `json:"context"`
	Status      string             `json:"status"`
	AdminNote   *string            `json:"admin_note,omitempty"`
	ResolvedBy  *feedbackPersonJSON `json:"resolved_by,omitempty"`
	ResolvedAt  *string            `json:"resolved_at,omitempty"`
	Submitter   feedbackPersonJSON `json:"submitter"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

func (d Deps) handleAdminFeedbackGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		adminID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid feedback id.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		sub, err := pfrepo.GetByID(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load feedback.")
			return
		}
		if sub == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Feedback not found.")
			return
		}
		detail, err := d.feedbackDetailJSON(r.Context(), sub)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load feedback.")
			return
		}
		d.recordFeedbackAdminAudit(r, adminID, auditservice.EventFeedbackAdminRead, &id, map[string]any{
			"action": "get",
			"id":     id.String(),
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(detail)
	}
}

type patchFeedbackBody struct {
	Status    *string `json:"status"`
	AdminNote *string `json:"admin_note"`
}

func (d Deps) handleAdminFeedbackPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		adminID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid feedback id.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		before, err := pfrepo.GetByID(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load feedback.")
			return
		}
		if before == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Feedback not found.")
			return
		}
		var body patchFeedbackBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var status *pfmodel.Status
		if body.Status != nil {
			parsed, err := pfmodel.ParseStatus(*body.Status)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status.")
				return
			}
			status = &parsed
		}
		if status == nil && body.AdminNote == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No fields to update.")
			return
		}
		updated, err := pfrepo.UpdateAdmin(r.Context(), d.Pool, id, adminID, status, body.AdminNote)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update feedback.")
			return
		}
		detail, err := d.feedbackDetailJSON(r.Context(), updated)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load feedback.")
			return
		}
		d.recordFeedbackAdminAudit(r, adminID, auditservice.EventFeedbackAdminUpdate, &id, map[string]any{
			"before": map[string]any{"status": before.Status, "admin_note": before.AdminNote},
			"after":  map[string]any{"status": updated.Status, "admin_note": updated.AdminNote},
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(detail)
	}
}

func (d Deps) feedbackDetailJSON(ctx context.Context, sub *pfrepo.Submission) (feedbackDetailJSON, error) {
	out := feedbackDetailJSON{
		ID:         sub.ID.String(),
		Message:    sub.Message,
		Category:   string(sub.Category),
		Source:     string(sub.Source),
		AppVersion: sub.AppVersion,
		Context:    sub.Context,
		Status:     string(sub.Status),
		AdminNote:  sub.AdminNote,
		CreatedAt:  sub.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  sub.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if sub.UserID != nil {
		s := sub.UserID.String()
		out.UserID = &s
		info, err := pfrepo.LookupSubmitter(ctx, d.Pool, *sub.UserID)
		if err != nil {
			return out, err
		}
		out.Submitter = feedbackPersonJSON{Name: info.Name, Email: info.Email}
	}
	if sub.OrgID != nil {
		s := sub.OrgID.String()
		out.OrgID = &s
	}
	if sub.ResolvedBy != nil {
		resolver, err := pfrepo.LookupResolver(ctx, d.Pool, *sub.ResolvedBy)
		if err != nil {
			return out, err
		}
		if resolver != nil {
			out.ResolvedBy = &feedbackPersonJSON{Name: resolver.Name, Email: resolver.Email}
		}
	}
	if sub.ResolvedAt != nil {
		s := sub.ResolvedAt.UTC().Format(time.RFC3339)
		out.ResolvedAt = &s
	}
	return out, nil
}

func (d Deps) recordFeedbackAdminAudit(r *http.Request, actorID uuid.UUID, eventType string, targetID *uuid.UUID, payload map[string]any) {
	if !d.effectiveConfig().AdminAuditLogEnabled || d.Pool == nil {
		return
	}
	orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
	var oid *uuid.UUID
	if orgID != uuid.Nil {
		oid = &orgID
	}
	tt := "product_feedback"
	after, _ := json.Marshal(payload)
	ip := clientIP(r)
	ua := r.UserAgent()
	_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
		OrgID:      oid,
		EventType:  eventType,
		ActorID:    actorID,
		ActorIP:    &ip,
		UserAgent:  &ua,
		TargetType: &tt,
		TargetID:   targetID,
		AfterValue: after,
	})
}
