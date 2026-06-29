package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	impersonationsvc "github.com/lextures/lextures/server/internal/service/impersonation"
)

func (d Deps) impersonationEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().ImpersonationEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Impersonation is not enabled.")
		return false
	}
	if !d.adminConsoleEnabled(w) {
		return false
	}
	return true
}

func (d Deps) registerImpersonationRoutes(r chi.Router) {
	r.Post("/api/v1/admin-console/impersonate", d.handleAdminConsoleImpersonateStart())
	r.Delete("/api/v1/admin-console/impersonate/session", d.handleAdminConsoleImpersonateEnd())
	r.Get("/api/v1/admin-console/impersonate/log", d.handleAdminConsoleImpersonateLog())
}

func (d Deps) impersonationWriteBlockMiddleware() func(http.Handler) http.Handler {
	const exitPath = "/api/v1/admin-console/impersonate/session"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := r.Method
			if m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			if m == http.MethodDelete && r.URL.Path == exitPath {
				next.ServeHTTP(w, r)
				return
			}
			if d.JWTSigner == nil {
				next.ServeHTTP(w, r)
				return
			}
			token, ok := auth.BearerToken(r.Header)
			if !ok || auth.JWTType(token) != "impersonation" {
				next.ServeHTTP(w, r)
				return
			}
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "writes_blocked_during_impersonation")
		})
	}
}

func (d Deps) handleAdminConsoleImpersonateStart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.impersonationEnabled(w) {
			return
		}
		if token, ok := auth.BearerToken(r.Header); ok && auth.JWTType(token) == "impersonation" {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Nested impersonation is not allowed.")
			return
		}
		// Require a normal login JWT (not an impersonation token).
		actor, ok := d.meSessionUserID(w, r)
		if !ok {
			return
		}
		_, targetOrg, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}

		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			TargetUserID string `json:"target_user_id"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		targetID, err := uuid.Parse(strings.TrimSpace(body.TargetUserID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid target_user_id.")
			return
		}
		userOrg, err := impersonationsvc.LookupTargetOrg(r.Context(), d.Pool, targetID)
		if err != nil {
			if errors.Is(err, impersonationsvc.ErrTargetNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load user.")
			return
		}
		if userOrg != targetOrg {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}

		ip := impersonationsvc.ClientIP(r)
		ua := r.UserAgent()
		result, err := impersonationsvc.Start(r.Context(), d.Pool, d.JWTSigner, d.effectiveConfig().AdminAuditLogEnabled, impersonationsvc.StartParams{
			ActorID:      actor,
			TargetUserID: targetID,
			TargetOrgID:  targetOrg,
			ActorIP:      &ip,
			UserAgent:    &ua,
		})
		if err != nil {
			switch {
			case errors.Is(err, impersonationsvc.ErrForbidden):
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			case errors.Is(err, impersonationsvc.ErrPrivilegedTarget):
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Cannot impersonate admin users.")
			case errors.Is(err, impersonationsvc.ErrTargetNotFound):
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			case errors.Is(err, impersonationsvc.ErrStoreDown):
				apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Impersonation is temporarily unavailable.")
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start impersonation.")
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"impersonation_token": result.Token,
			"expires_at":          result.ExpiresAt,
			"target":              result.Target,
		})
	}
}

func (d Deps) handleAdminConsoleImpersonateEnd() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.impersonationEnabled(w) {
			return
		}
		if d.JWTSigner == nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
			return
		}
		token, ok := auth.BearerToken(r.Header)
		if !ok || auth.JWTType(token) != "impersonation" {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Not in an impersonation session.")
			return
		}
		imp, err := d.JWTSigner.VerifyImpersonation(token)
		if err != nil {
			if errors.Is(err, auth.ErrExpiredToken) {
				apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Impersonation session expired.")
				return
			}
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid impersonation session.")
			return
		}
		adminID, err := uuid.Parse(imp.AdminID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid impersonation session.")
			return
		}
		targetID, err := uuid.Parse(imp.TargetUserID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Invalid impersonation session.")
			return
		}
		targetOrg, err := impersonationsvc.LookupTargetOrg(r.Context(), d.Pool, targetID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to end impersonation.")
			return
		}
		ip := impersonationsvc.ClientIP(r)
		ua := r.UserAgent()
		if err := impersonationsvc.End(r.Context(), d.Pool, d.effectiveConfig().AdminAuditLogEnabled, impersonationsvc.EndParams{
			JTI:          imp.JTI,
			AdminID:      adminID,
			TargetUserID: targetID,
			TargetOrgID:  targetOrg,
			ActorIP:      &ip,
			UserAgent:    &ua,
		}); err != nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Failed to end impersonation.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleAdminConsoleImpersonateLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.impersonationEnabled(w) {
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		if !d.effectiveConfig().AdminAuditLogEnabled {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Admin audit log is not enabled.")
			return
		}
		q := r.URL.Query()
		eventType := auditservice.EventUserImpersonation
		params := auditservice.QueryParams{OrgID: &orgID, EventType: &eventType, Limit: 100}
		if s := strings.TrimSpace(q.Get("target_user_id")); s != "" {
			tid, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid target_user_id.")
				return
			}
			params.TargetID = &tid
		}
		events, err := auditservice.ListEvents(r.Context(), d.Pool, params)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load impersonation log.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": auditEventsToJSON(events)})
	}
}
