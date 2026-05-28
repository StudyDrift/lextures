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
	repoDR "github.com/lextures/lextures/server/internal/repos/dataresidency"
	"github.com/lextures/lextures/server/internal/repos/organization"
	drservice "github.com/lextures/lextures/server/internal/service/dataresidency"
)

func (d Deps) dataResidencyEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().DataResidencyEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Data residency module is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireDataResidencyAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	isAdmin, err := drservice.CheckAdmin(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
		return uuid.UUID{}, false
	}
	if !isAdmin {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	return uid, true
}

func (d Deps) registerDataResidencyRoutes(r chi.Router) {
	r.Get("/api/v1/internal/compliance/data-residency/org/{orgId}", d.handleGetOrgDataResidency())
	r.Get("/api/v1/internal/compliance/data-residency/access-log", d.handleGetDataResidencyAccessLog())
}

// GET /api/v1/internal/compliance/data-residency/org/{orgId}
func (d Deps) handleGetOrgDataResidency() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.dataResidencyEnabled(w) {
			return
		}
		if _, ok := d.requireDataResidencyAdmin(w, r); !ok {
			return
		}
		idStr := strings.TrimSpace(chi.URLParam(r, "orgId"))
		id, err := uuid.Parse(idStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		org, err := organization.GetByID(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load organization.")
			return
		}
		if org == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Organization not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"orgId":      org.ID.String(),
			"orgName":    org.Name,
			"dataRegion": org.DataRegion,
			"createdAt":  org.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
}

// GET /api/v1/internal/compliance/data-residency/access-log
func (d Deps) handleGetDataResidencyAccessLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.dataResidencyEnabled(w) {
			return
		}
		if _, ok := d.requireDataResidencyAdmin(w, r); !ok {
			return
		}
		ctx := r.Context()
		limit := int32(100)
		offset := int32(0)
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.ParseInt(v, 10, 32); err == nil && n > 0 {
				limit = int32(n)
			}
		}
		if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
			if n, err := strconv.ParseInt(v, 10, 32); err == nil && n >= 0 {
				offset = int32(n)
			}
		}

		var entries []repoDR.AccessLogEntry
		var err error

		if orgStr := strings.TrimSpace(r.URL.Query().Get("orgId")); orgStr != "" {
			orgID, parseErr := uuid.Parse(orgStr)
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
				return
			}
			entries, err = repoDR.ListAccessLogByOrg(ctx, d.Pool, orgID, limit, offset)
		} else {
			entries, err = repoDR.ListAccessLog(ctx, d.Pool, limit, offset)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load access log.")
			return
		}

		out := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			m := map[string]any{
				"id":            e.ID.String(),
				"orgId":         e.OrgID.String(),
				"orgRegion":     e.OrgRegion,
				"requestedFrom": e.RequestedFrom,
				"eventType":     e.EventType,
				"createdAt":     e.CreatedAt.UTC().Format(time.RFC3339),
			}
			if e.RequestPath != nil {
				m["requestPath"] = *e.RequestPath
			}
			if e.ActorID != nil {
				m["actorId"] = e.ActorID.String()
			}
			out = append(out, m)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})
	}
}
