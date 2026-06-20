package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/api"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/apitokens"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

func (d Deps) requireOrgTokenAdmin(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := d.meSessionUserID(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	if !d.requireAPITokensEnabled(w) {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	isGA, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, permGlobalRBACManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	if isGA {
		return userID, orgID, true
	}
	isAdmin, err := orgroles.UserHasRole(r.Context(), d.Pool, userID, orgID, orgroles.RoleOrgAdmin)
	if err != nil || !isAdmin {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return userID, orgID, true
}

func (d Deps) handleAdminListTokens() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, orgID, ok := d.requireOrgTokenAdmin(w, r)
		if !ok {
			return
		}
		rows, err := apitokens.ListByOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load access keys.")
			return
		}
		now := time.Now().UTC()
		items := make([]accessKeyItem, 0, len(rows))
		for _, row := range rows {
			var summaries []course.Summary
			if len(row.CourseIDs) > 0 {
				summaries, err = course.SummariesByIDs(r.Context(), d.Pool, row.CourseIDs)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load access keys.")
					return
				}
			}
			courses := make([]accessKeyCourse, 0, len(summaries))
			for _, s := range summaries {
				courses = append(courses, accessKeyCourse{
					ID:         s.ID.String(),
					CourseCode: s.CourseCode,
					Title:      s.Title,
				})
			}
			items = append(items, accessKeyItemFromRow(row, courses, now))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"tokens": items})
	}
}

type postServiceTokenBody struct {
	Label              string   `json:"label"`
	ServiceAccountName string   `json:"serviceAccountName"`
	Scopes             []string `json:"scopes"`
	ExpiresAt          *string  `json:"expiresAt"`
}

func (d Deps) handleAdminPostServiceToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actorID, orgID, ok := d.requireOrgTokenAdmin(w, r)
		if !ok {
			return
		}
		var body postServiceTokenBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		scopes, valid := api.NormalizeScopes(body.Scopes)
		if !valid || len(scopes) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Select at least one valid scope.")
			return
		}
		expiresAt, err := parseOptionalExpiresAt(body.ExpiresAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		row, secret, err := apitokens.InsertService(
			r.Context(), d.Pool, orgID,
			strings.TrimSpace(body.ServiceAccountName),
			strings.TrimSpace(body.Label),
			scopes,
			expiresAt,
		)
		if err != nil {
			if err.Error() == "maximum number of service tokens reached" || err.Error() == "service account name required" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not create service token.")
			return
		}
		apitokens.AuditCreate(r.Context(), d.Pool, actorID, &orgID, row.ID, scopes, r)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                 row.ID.String(),
			"token":              secret,
			"label":              row.Label,
			"serviceAccountName": row.ServiceAccountName,
			"scopes":             row.Scopes,
			"expiresAt":          row.ExpiresAt,
			"createdAt":          row.CreatedAt,
		})
	}
}

func (d Deps) handleAdminDeleteToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actorID, orgID, ok := d.requireOrgTokenAdmin(w, r)
		if !ok {
			return
		}
		raw := chi.URLParam(r, "id")
		tokenID, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid access key id.")
			return
		}
		row, err := apitokens.GetByID(r.Context(), d.Pool, tokenID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Access key not found.")
			return
		}
		tokenOrg, err := apitokens.OrgIDForToken(r.Context(), d.Pool, tokenID)
		if err != nil || tokenOrg == nil || *tokenOrg != orgID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Access key not found.")
			return
		}
		okRevoke, err := apitokens.RevokeByID(r.Context(), d.Pool, tokenID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not revoke access key.")
			return
		}
		if !okRevoke {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Access key not found.")
			return
		}
		apitokens.AuditRevoke(r.Context(), d.Pool, actorID, &orgID, tokenID, r)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

func (d Deps) registerAdminTokenRoutes(r chi.Router) {
	r.Get("/api/v1/admin/tokens", d.handleAdminListTokens())
	r.Post("/api/v1/admin/tokens", d.handleAdminPostServiceToken())
	r.Delete("/api/v1/admin/tokens/{id}", d.handleAdminDeleteToken())
}
