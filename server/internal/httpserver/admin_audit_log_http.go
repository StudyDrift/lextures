package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoadminaudit "github.com/lextures/lextures/server/internal/repos/adminaudit"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
)

func (d Deps) adminAuditLogEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().AdminAuditLogEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Admin audit log is not enabled.")
		return false
	}
	return true
}

// requireAuditReadAccess authenticates the caller and enforces compliance:audit:read:*.
func (d Deps) requireAuditReadAccess(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	canRead, err := auditservice.CheckReadAccess(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
		return uuid.UUID{}, false
	}
	if !canRead {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to access the audit log.")
		return uuid.UUID{}, false
	}
	return uid, true
}

func (d Deps) registerAdminAuditLogRoutes(r chi.Router) {
	r.Get("/api/v1/compliance/audit-log", d.handleGetAuditLog())
	r.Get("/api/v1/compliance/audit-log/export", d.handleGetAuditLogExport())
	r.Get("/api/v1/compliance/audit-log/{event_id}", d.handleGetAuditLogEvent())
}

// GET /api/v1/compliance/audit-log
func (d Deps) handleGetAuditLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.adminAuditLogEnabled(w) {
			return
		}
		if _, ok := d.requireAuditReadAccess(w, r); !ok {
			return
		}
		q := r.URL.Query()
		from, to := parseTimeWindow(q.Get("from"), q.Get("to"))

		params := auditservice.QueryParams{From: from, To: to, Limit: 500}
		if s := strings.TrimSpace(q.Get("actorId")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid actorId.")
				return
			}
			params.ActorID = &id
		}
		if s := strings.TrimSpace(q.Get("orgId")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
				return
			}
			params.OrgID = &id
		}
		if s := strings.TrimSpace(q.Get("targetId")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid targetId.")
				return
			}
			params.TargetID = &id
		}
		if s := strings.TrimSpace(q.Get("eventType")); s != "" {
			params.EventType = &s
		}

		events, err := auditservice.ListEvents(r.Context(), d.Pool, params)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load audit log.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"events": auditEventsToJSON(events)})
	}
}

// GET /api/v1/compliance/audit-log/export
func (d Deps) handleGetAuditLogExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.adminAuditLogEnabled(w) {
			return
		}
		if _, ok := d.requireAuditReadAccess(w, r); !ok {
			return
		}
		q := r.URL.Query()
		from, to := parseTimeWindow(q.Get("from"), q.Get("to"))
		format := strings.ToLower(strings.TrimSpace(q.Get("format")))

		params := auditservice.QueryParams{From: from, To: to, Limit: 100000}
		if s := strings.TrimSpace(q.Get("orgId")); s != "" {
			if id, err := uuid.Parse(s); err == nil {
				params.OrgID = &id
			}
		}
		if s := strings.TrimSpace(q.Get("actorId")); s != "" {
			if id, err := uuid.Parse(s); err == nil {
				params.ActorID = &id
			}
		}
		if s := strings.TrimSpace(q.Get("eventType")); s != "" {
			params.EventType = &s
		}

		events, err := auditservice.ListEvents(r.Context(), d.Pool, params)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not export audit log.")
			return
		}

		if format == "csv" {
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="admin_audit_log.csv"`)
			cw := csv.NewWriter(w)
			_ = cw.Write([]string{
				"event_id", "timestamp", "event_type", "actor_id", "actor_ip",
				"org_id", "target_type", "target_id", "before_value", "after_value",
			})
			for _, e := range events {
				_ = cw.Write([]string{
					e.EventID.String(),
					e.Timestamp.UTC().Format(time.RFC3339Nano),
					e.EventType,
					e.ActorID.String(),
					strOrEmpty(e.ActorIP),
					uuidOrEmpty(e.OrgID),
					strOrEmpty(e.TargetType),
					uuidOrEmpty(e.TargetID),
					string(e.BeforeValue),
					string(e.AfterValue),
				})
			}
			cw.Flush()
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"events": auditEventsToJSON(events)})
	}
}

// GET /api/v1/compliance/audit-log/{event_id}
func (d Deps) handleGetAuditLogEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.adminAuditLogEnabled(w) {
			return
		}
		if _, ok := d.requireAuditReadAccess(w, r); !ok {
			return
		}
		eventID, err := uuid.Parse(chi.URLParam(r, "event_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid event_id.")
			return
		}
		e, err := auditservice.GetEvent(r.Context(), d.Pool, eventID)
		if err != nil {
			if errors.Is(err, auditservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Audit event not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load audit event.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(auditEventToJSON(*e))
	}
}

func auditEventToJSON(e repoadminaudit.Event) map[string]any {
	m := map[string]any{
		"eventId":   e.EventID.String(),
		"eventType": e.EventType,
		"actorId":   e.ActorID.String(),
		"timestamp": e.Timestamp.UTC().Format(time.RFC3339Nano),
	}
	if e.OrgID != nil {
		m["orgId"] = e.OrgID.String()
	}
	if e.ActorIP != nil {
		m["actorIp"] = *e.ActorIP
	}
	if e.UserAgent != nil {
		m["userAgent"] = *e.UserAgent
	}
	if e.TargetType != nil {
		m["targetType"] = *e.TargetType
	}
	if e.TargetID != nil {
		m["targetId"] = e.TargetID.String()
	}
	if len(e.BeforeValue) > 0 {
		m["beforeValue"] = json.RawMessage(e.BeforeValue)
	}
	if len(e.AfterValue) > 0 {
		m["afterValue"] = json.RawMessage(e.AfterValue)
	}
	if e.ChainHash != nil {
		m["chainHash"] = *e.ChainHash
	}
	return m
}

func auditEventsToJSON(events []repoadminaudit.Event) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	for _, e := range events {
		out = append(out, auditEventToJSON(e))
	}
	return out
}

func uuidOrEmpty(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}
