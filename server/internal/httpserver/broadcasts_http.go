package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoBroadcasts "github.com/lextures/lextures/server/internal/repos/broadcasts"
)

const (
	broadcastTypeAnnouncement = "announcement"
	broadcastTypeEmergency    = "emergency"
)

func (d Deps) handleOrgBroadcasts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Config.FFBroadcasts {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Broadcasts feature is not enabled.")
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}

		switch r.Method {
		case http.MethodGet:
			if _, ok := d.orgRoleAccess(w, r, orgID, false); !ok {
				return
			}
			limit := 100
			if l := strings.TrimSpace(r.URL.Query().Get("limit")); l != "" {
				if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
					limit = n
				}
			}
			items, err := repoBroadcasts.ListByOrg(r.Context(), d.Pool, orgID, limit)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list broadcasts.")
				return
			}
			out := make([]map[string]any, 0, len(items))
			for i := range items {
				out = append(out, broadcastToJSON(&items[i]))
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"broadcasts": out})

		case http.MethodPost:
			actor, ok := d.orgRoleAccess(w, r, orgID, true)
			if !ok {
				return
			}
			var body struct {
				Type        string          `json:"type"`
				SchoolID    *string         `json:"schoolId"`
				Audience    json.RawMessage `json:"audience"`
				Subject     string          `json:"subject"`
				Body        string          `json:"body"`
				ScheduledAt *string         `json:"scheduledAt"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			subject := strings.TrimSpace(body.Subject)
			text := strings.TrimSpace(body.Body)
			if subject == "" || text == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "subject and body are required.")
				return
			}
			bType := strings.TrimSpace(body.Type)
			if bType == "" {
				bType = broadcastTypeAnnouncement
			}
			if bType != broadcastTypeAnnouncement && bType != broadcastTypeEmergency {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "type must be announcement or emergency.")
				return
			}
			var schoolUUID *uuid.UUID
			if body.SchoolID != nil && strings.TrimSpace(*body.SchoolID) != "" {
				parsed, err := uuid.Parse(strings.TrimSpace(*body.SchoolID))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid schoolId.")
					return
				}
				schoolUUID = &parsed
			}
			var scheduledAt *time.Time
			status := "sent"
			if body.ScheduledAt != nil && strings.TrimSpace(*body.ScheduledAt) != "" {
				t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.ScheduledAt))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scheduledAt must be RFC3339.")
					return
				}
				if t.After(time.Now().Add(7 * 24 * time.Hour)) {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scheduledAt cannot be more than 7 days out.")
					return
				}
				scheduledAt = &t
				status = "queued"
			}
			b, err := repoBroadcasts.Create(r.Context(), d.Pool, repoBroadcasts.CreateParams{
				OrgID:       orgID,
				SchoolID:    schoolUUID,
				SenderID:    actor,
				Type:        bType,
				Audience:    body.Audience,
				Subject:     subject,
				Body:        text,
				ScheduledAt: scheduledAt,
				Status:      status,
			})
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create broadcast.")
				return
			}
			if status == "sent" {
				if _, err := repoBroadcasts.EnqueueRecipients(r.Context(), d.Pool, b.ID, orgID); err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue recipients.")
					return
				}
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"broadcast": broadcastToJSON(b)})

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

func (d Deps) handleOrgBroadcastDeliveryReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Config.FFBroadcasts {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Broadcasts feature is not enabled.")
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		broadcastID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "broadcastId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid broadcast id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, false); !ok {
			return
		}
		b, err := repoBroadcasts.Get(r.Context(), d.Pool, orgID, broadcastID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load broadcast.")
			return
		}
		if b == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Broadcast not found.")
			return
		}
		rpt, err := repoBroadcasts.GetDeliveryReport(r.Context(), d.Pool, broadcastID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load delivery report.")
			return
		}
		unack := make([]map[string]any, 0, len(rpt.Unacknowledged))
		for _, u := range rpt.Unacknowledged {
			m := map[string]any{"userId": u.UserID.String(), "email": u.Email}
			if u.DisplayName != nil {
				m["displayName"] = *u.DisplayName
			} else {
				m["displayName"] = nil
			}
			unack = append(unack, m)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"broadcastId":     b.ID.String(),
			"totalRecipients": rpt.TotalRecipients,
			"acknowledged":    rpt.Acknowledged,
			"unacknowledged":  unack,
		})
	}
}

func (d Deps) handleBroadcastAcknowledge() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Config.FFBroadcasts {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Broadcasts feature is not enabled.")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		broadcastID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "broadcastId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid broadcast id.")
			return
		}
		actor, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if err := repoBroadcasts.Acknowledge(r.Context(), d.Pool, broadcastID, actor); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record acknowledgement.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleMeBroadcasts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Config.FFBroadcasts {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Broadcasts feature is not enabled.")
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		items, err := repoBroadcasts.ListForUser(r.Context(), d.Pool, actor)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load broadcasts.")
			return
		}
		out := make([]map[string]any, 0, len(items))
		for i := range items {
			out = append(out, broadcastToJSON(&items[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"broadcasts": out})
	}
}

func (d Deps) registerBroadcastRoutes(r chi.Router) {
	r.Method(http.MethodGet, "/api/v1/orgs/{orgId}/broadcasts", d.handleOrgBroadcasts())
	r.Method(http.MethodPost, "/api/v1/orgs/{orgId}/broadcasts", d.handleOrgBroadcasts())
	r.Method(http.MethodGet, "/api/v1/orgs/{orgId}/broadcasts/{broadcastId}/delivery-report", d.handleOrgBroadcastDeliveryReport())
	r.Method(http.MethodPost, "/api/v1/broadcasts/{broadcastId}/acknowledge", d.handleBroadcastAcknowledge())
	r.Method(http.MethodGet, "/api/v1/me/broadcasts", d.handleMeBroadcasts())
}

func broadcastToJSON(b *repoBroadcasts.Broadcast) map[string]any {
	if b == nil {
		return nil
	}
	m := map[string]any{
		"id":        b.ID.String(),
		"orgId":     b.OrgID.String(),
		"senderId":  b.SenderID.String(),
		"type":      b.Type,
		"subject":   b.Subject,
		"body":      b.Body,
		"status":    b.Status,
		"createdAt": b.CreatedAt.UTC().Format(time.RFC3339),
	}
	if b.SchoolID != nil {
		m["schoolId"] = b.SchoolID.String()
	} else {
		m["schoolId"] = nil
	}
	if len(b.Audience) > 0 {
		m["audience"] = json.RawMessage(b.Audience)
	} else {
		m["audience"] = json.RawMessage("{}")
	}
	if b.ScheduledAt != nil {
		m["scheduledAt"] = b.ScheduledAt.UTC().Format(time.RFC3339)
	} else {
		m["scheduledAt"] = nil
	}
	if b.SentAt != nil {
		m["sentAt"] = b.SentAt.UTC().Format(time.RFC3339)
	} else {
		m["sentAt"] = nil
	}
	return m
}

