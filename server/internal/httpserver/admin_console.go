package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/adminconsole"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgbranding"
	"github.com/lextures/lextures/server/internal/repos/orgrolegrant"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	cfservice "github.com/lextures/lextures/server/internal/service/customfields"
	"github.com/lextures/lextures/server/internal/service/licensesvc"
)

func (d Deps) adminConsoleEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().AdminConsoleEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Admin console is not enabled.")
		return false
	}
	return true
}

// adminConsoleAccess authenticates the caller and ensures org_admin or global admin for targetOrg.
// Global admins may pass ?orgId= to scope to another organization.
// Requires the Admin Console platform feature to be enabled.
func (d Deps) adminConsoleAccess(w http.ResponseWriter, r *http.Request, wantManage bool) (actor uuid.UUID, targetOrg uuid.UUID, globalAdmin bool, ok bool) {
	if !d.adminConsoleEnabled(w) {
		return uuid.UUID{}, uuid.UUID{}, false, false
	}
	return d.orgAdminAccess(w, r, wantManage)
}

// orgAdminAccess is the same permission model as adminConsoleAccess (org admin / global admin)
// but does not require the Admin Console feature flag. Used by features that share console
// permissions but ship independently (e.g. maintenance banners, plan 18.6).
func (d Deps) orgAdminAccess(w http.ResponseWriter, r *http.Request, wantManage bool) (actor uuid.UUID, targetOrg uuid.UUID, globalAdmin bool, ok bool) {
	actor, ok = d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false, false
	}
	ctx := r.Context()

	ga, err := rbac.UserHasPermission(ctx, d.Pool, actor, permGlobalRBACManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, uuid.UUID{}, false, false
	}

	targetOrg, ok = d.resolveAdminConsoleOrgID(w, r, actor, ga)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false, false
	}

	if ga {
		return actor, targetOrg, true, true
	}

	uOrg, err := organization.OrgIDForUser(ctx, d.Pool, actor)
	if err != nil || uOrg != targetOrg {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, uuid.UUID{}, false, false
	}

	if wantManage {
		has, err := orgroles.UserHasRole(ctx, d.Pool, actor, targetOrg, orgroles.RoleOrgAdmin)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return uuid.UUID{}, uuid.UUID{}, false, false
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return uuid.UUID{}, uuid.UUID{}, false, false
		}
		return actor, targetOrg, false, true
	}

	admin, err := orgroles.UserHasRole(ctx, d.Pool, actor, targetOrg, orgroles.RoleOrgAdmin)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, uuid.UUID{}, false, false
	}
	if admin {
		return actor, targetOrg, false, true
	}
	viewer, err := orgroles.UserHasRole(ctx, d.Pool, actor, targetOrg, orgroles.RoleOrgViewer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, uuid.UUID{}, false, false
	}
	if !viewer {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, uuid.UUID{}, false, false
	}
	return actor, targetOrg, false, true
}

func (d Deps) resolveAdminConsoleOrgID(w http.ResponseWriter, r *http.Request, actor uuid.UUID, globalAdmin bool) (uuid.UUID, bool) {
	ctx := r.Context()
	if s := strings.TrimSpace(r.URL.Query().Get("orgId")); s != "" {
		oid, err := uuid.Parse(s)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
			return uuid.UUID{}, false
		}
		if !globalAdmin {
			uOrg, err := organization.OrgIDForUser(ctx, d.Pool, actor)
			if err != nil || uOrg != oid {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
				return uuid.UUID{}, false
			}
		}
		return oid, true
	}
	uOrg, err := organization.OrgIDForUser(ctx, d.Pool, actor)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load organization.")
		return uuid.UUID{}, false
	}
	return uOrg, true
}

func (d Deps) registerAdminConsoleRoutes(r chi.Router) {
	r.Get("/api/v1/admin-console/overview", d.handleAdminConsoleOverview())
	r.Get("/api/v1/admin-console/users", d.handleAdminConsoleUsers())
	r.Get("/api/v1/admin-console/users/export.csv", d.handleAdminConsoleUsersExport())
	r.Get("/api/v1/admin-console/users/{userId}", d.handleAdminConsoleUserGet())
	r.Patch("/api/v1/admin-console/users/{userId}", d.handleAdminConsoleUserPatch())
	r.Get("/api/v1/admin-console/courses", d.handleAdminConsoleCourses())
	r.Patch("/api/v1/admin-console/courses/{courseId}/status", d.handleAdminConsoleCourseStatusPatch())
	r.Get("/api/v1/admin-console/audit-log", d.handleAdminConsoleAuditLog())
	r.Get("/api/v1/admin-console/settings", d.handleAdminConsoleSettings())
	r.Put("/api/v1/admin-console/settings", d.handleAdminConsoleSettings())
	r.Post("/api/v1/admin-console/delegate", d.handleAdminConsoleDelegate())
	r.Get("/api/v1/admin-console/integrations", d.handleAdminConsoleIntegrations())
	r.Get("/api/v1/me/admin-console-capabilities", d.handleMeAdminConsoleCapabilities())
	d.registerAdminImportRoutes(r)
	d.registerAdminEmailTemplateRoutes(r)
	d.registerAdminCustomFieldRoutes(r)
	d.registerAdminLicenseRoutes(r)
}

func (d Deps) handleMeAdminConsoleCapabilities() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		enabled := d.effectiveConfig().AdminConsoleEnabled
		ctx := r.Context()
		orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load organization.")
			return
		}
		ga, err := rbac.UserHasPermission(ctx, d.Pool, userID, permGlobalRBACManage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		canAccess := false
		canManage := ga
		if enabled {
			if ga {
				canAccess = true
			} else {
				admin, err := orgroles.UserHasRole(ctx, d.Pool, userID, orgID, orgroles.RoleOrgAdmin)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
					return
				}
				viewer, err := orgroles.UserHasRole(ctx, d.Pool, userID, orgID, orgroles.RoleOrgViewer)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
					return
				}
				canAccess = admin || viewer
				canManage = admin
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":             enabled,
			"orgId":               orgID.String(),
			"canAccess":           canAccess,
			"canManage":           canManage,
			"isGlobalAdmin":       ga,
			"customFieldsEnabled": d.effectiveConfig().CustomFieldsEnabled,
			"seatManagementEnabled": d.effectiveConfig().SeatManagementEnabled,
		})
	}
}

func (d Deps) handleAdminConsoleOverview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		overview, err := adminconsole.OverviewForOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load overview.")
			return
		}

		var recent []map[string]any
		if d.effectiveConfig().AdminAuditLogEnabled {
			events, err := auditservice.ListEvents(r.Context(), d.Pool, auditservice.QueryParams{
				OrgID: &orgID,
				Limit: 20,
			})
			if err == nil {
				recent = auditEventsToJSON(events)
			}
		}
		if recent == nil {
			recent = []map[string]any{}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"orgId":            orgID.String(),
			"totalUsers":       overview.TotalUsers,
			"activeCourses":    overview.ActiveCourses,
			"pendingEnrollments": overview.PendingEnrollments,
			"storageBytes":     overview.StorageBytes,
			"recentAuditEvents": recent,
			"license":          d.licenseOverviewJSON(r, orgID),
		})
	}
}

func parseAdminConsoleListParams(r *http.Request) adminconsole.ListParams {
	p := adminconsole.ListParams{
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
		Role:  strings.TrimSpace(r.URL.Query().Get("role")),
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
	}
	if v := strings.TrimSpace(r.URL.Query().Get("page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.Page = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("per_page")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.PerPage = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("perPage")); v != "" && p.PerPage == 0 {
		if n, err := strconv.Atoi(v); err == nil {
			p.PerPage = n
		}
	}
	if s := strings.TrimSpace(r.URL.Query().Get("term_id")); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			p.TermID = &id
		}
	}
	if s := strings.TrimSpace(r.URL.Query().Get("termId")); s != "" && p.TermID == nil {
		if id, err := uuid.Parse(s); err == nil {
			p.TermID = &id
		}
	}
	return p
}

func (d Deps) handleAdminConsoleUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		result, err := adminconsole.ListUsers(r.Context(), d.Pool, orgID, parseAdminConsoleListParams(r))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list users.")
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (d Deps) handleAdminConsoleUserPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		targetID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "userId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			Active       *bool          `json:"active"`
			Role         *string        `json:"role"`
			CustomFields map[string]any `json:"customFields"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		ctx := r.Context()
		var targetOrg uuid.UUID
		err = d.Pool.QueryRow(ctx, `SELECT org_id FROM "user".users WHERE id = $1`, targetID).Scan(&targetOrg)
		if err != nil {
			if err == pgx.ErrNoRows {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load user.")
			return
		}
		if targetOrg != orgID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}

		if body.Active != nil && !*body.Active {
			tag, err := d.Pool.Exec(ctx, `
UPDATE "user".users
SET deactivated_at = COALESCE(deactivated_at, NOW()), login_blocked = TRUE
WHERE id = $1 AND org_id = $2
`, targetID, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to deactivate user.")
				return
			}
			if tag.RowsAffected() == 0 {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
				return
			}
			d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventUserDeactivate, "user", &targetID, nil, raw)
		} else if body.Active != nil && *body.Active {
			if err := d.licenseService().CheckCanActivate(ctx, targetID, orgID); err != nil {
				if errors.Is(err, licensesvc.ErrSeatLimitReached) {
					writeSeatLimitError(w)
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify seat license.")
				return
			}
			_, err := d.Pool.Exec(ctx, `
UPDATE "user".users SET deactivated_at = NULL, login_blocked = FALSE WHERE id = $1 AND org_id = $2
`, targetID, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reactivate user.")
				return
			}
		}

		if body.Role != nil {
			dbRole := cliRoleToAppRole(strings.TrimSpace(*body.Role))
			var roleID string
			err := d.Pool.QueryRow(ctx, `SELECT id::text FROM "user".app_roles WHERE name = $1`, dbRole).Scan(&roleID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown role.")
				return
			}
			_, _ = d.Pool.Exec(ctx, `DELETE FROM "user".user_app_roles WHERE user_id = $1`, targetID)
			_, err = d.Pool.Exec(ctx, `
INSERT INTO "user".user_app_roles (user_id, role_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, targetID, roleID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update role.")
				return
			}
			d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventRoleGrant, "user", &targetID, nil, raw)
		}

		if body.CustomFields != nil {
			if !d.effectiveConfig().CustomFieldsEnabled {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Custom fields are not enabled.")
				return
			}
			merged, valErrs, err := cfservice.New(d.Pool).SetUserValues(ctx, orgID, targetID, body.CustomFields)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update custom fields.")
				return
			}
			if len(valErrs) > 0 {
				writeCustomFieldValidationErrors(w, valErrs)
				return
			}
			d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventUserUpdate, "user", &targetID, nil, raw)
			if wantsInclude(r, "custom_fields") {
				writeJSON(w, http.StatusOK, map[string]any{"id": targetID.String(), "customFields": merged})
				return
			}
		}

		result, err := adminconsole.ListUsers(ctx, d.Pool, orgID, adminconsole.ListParams{Page: 1, PerPage: 1, Query: targetID.String()})
		if err != nil || len(result.Items) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{"id": targetID.String()})
			return
		}
		writeJSON(w, http.StatusOK, result.Items[0])
	}
}

func (d Deps) handleAdminConsoleCourses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		result, err := adminconsole.ListCourses(r.Context(), d.Pool, orgID, parseAdminConsoleListParams(r))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (d Deps) handleAdminConsoleCourseStatusPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		courseID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "courseId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		ctx := r.Context()
		var courseCode string
		var courseOrg uuid.UUID
		err = d.Pool.QueryRow(ctx, `SELECT course_code, org_id FROM course.courses WHERE id = $1`, courseID).Scan(&courseCode, &courseOrg)
		if err != nil {
			if err == pgx.ErrNoRows {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if courseOrg != orgID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}

		switch strings.ToLower(strings.TrimSpace(body.Status)) {
		case "archived":
			_, err = course.SetArchived(ctx, d.Pool, courseCode, true, &actor)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to archive course.")
				return
			}
			d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventCourseArchive, "course", &courseID, nil, raw)
		case "active":
			_, err = course.SetArchived(ctx, d.Pool, courseCode, false, nil)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to restore course.")
				return
			}
			_, err = d.Pool.Exec(ctx, `UPDATE course.courses SET published = true WHERE id = $1`, courseID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to publish course.")
				return
			}
		case "draft":
			_, err = course.SetArchived(ctx, d.Pool, courseCode, false, nil)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update course.")
				return
			}
			_, err = d.Pool.Exec(ctx, `UPDATE course.courses SET published = false WHERE id = $1`, courseID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update course.")
				return
			}
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be active, archived, or draft.")
			return
		}

		result, err := adminconsole.ListCourses(ctx, d.Pool, orgID, adminconsole.ListParams{Page: 1, PerPage: 1, Query: courseCode})
		if err != nil || len(result.Items) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{"id": courseID.String(), "courseCode": courseCode})
			return
		}
		writeJSON(w, http.StatusOK, result.Items[0])
	}
}

func (d Deps) handleAdminConsoleAuditLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
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
		from, to := parseTimeWindow(q.Get("from"), q.Get("to"))
		params := auditservice.QueryParams{OrgID: &orgID, From: from, To: to, Limit: 500}
		if s := strings.TrimSpace(q.Get("action")); s != "" {
			params.EventType = &s
		}
		if s := strings.TrimSpace(q.Get("eventType")); s != "" && params.EventType == nil {
			params.EventType = &s
		}
		events, err := auditservice.ListEvents(r.Context(), d.Pool, params)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load audit log.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": auditEventsToJSON(events)})
	}
}

func (d Deps) handleAdminConsoleSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wantManage := r.Method == http.MethodPut
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, wantManage)
		if !ok {
			return
		}
		ctx := r.Context()
		switch r.Method {
		case http.MethodGet:
			writeAdminConsoleSettings(w, ctx, d.Pool, orgID)
		case http.MethodPut:
			raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
				return
			}
			var body struct {
				Name                   *string `json:"name"`
				LogoURL                *string `json:"logoUrl"`
				FaviconURL             *string `json:"faviconUrl"`
				PrimaryColor           *string `json:"primaryColor"`
				SecondaryColor         *string `json:"secondaryColor"`
				CustomDomain           *string `json:"customDomain"`
				CustomEmailDisplayName *string `json:"customEmailDisplayName"`
				Timezone               *string `json:"timezone"`
				Locale                 *string `json:"locale"`
			}
			if err := json.Unmarshal(raw, &body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			if body.Name != nil {
				_, err := organization.Patch(ctx, d.Pool, orgID, body.Name, nil, nil, nil, nil, nil)
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not update organization.")
					return
				}
			}
			if body.Timezone != nil || body.Locale != nil {
				var meta map[string]any
				org, _ := organization.GetByID(ctx, d.Pool, orgID)
				if org != nil {
					_ = json.Unmarshal(org.Metadata, &meta)
				}
				if meta == nil {
					meta = map[string]any{}
				}
				if body.Timezone != nil {
					meta["timezone"] = strings.TrimSpace(*body.Timezone)
				}
				if body.Locale != nil {
					meta["locale"] = strings.TrimSpace(*body.Locale)
				}
				metaBytes, _ := json.Marshal(meta)
				rawMeta := json.RawMessage(metaBytes)
				_, _ = organization.Patch(ctx, d.Pool, orgID, nil, nil, nil, nil, nil, &rawMeta)
			}
			cur, _ := orgbranding.Get(ctx, d.Pool, orgID)
			logo := mergeStrPtr(cur, func(row *orgbranding.Row) *string {
				if row == nil {
					return nil
				}
				return row.LogoURL
			}, body.LogoURL)
			fav := mergeStrPtr(cur, func(row *orgbranding.Row) *string {
				if row == nil {
					return nil
				}
				return row.FaviconURL
			}, body.FaviconURL)
			p1 := orgbranding.DefaultPrimaryHex
			if cur != nil && strings.TrimSpace(cur.PrimaryColor) != "" {
				p1 = cur.PrimaryColor
			}
			if body.PrimaryColor != nil {
				if v, err := orgbranding.ValidateHexColor(*body.PrimaryColor); err == nil {
					p1 = v
				}
			}
			p2 := orgbranding.DefaultSecondaryHex
			if cur != nil && strings.TrimSpace(cur.SecondaryColor) != "" {
				p2 = cur.SecondaryColor
			}
			if body.SecondaryColor != nil {
				if v, err := orgbranding.ValidateHexColor(*body.SecondaryColor); err == nil {
					p2 = v
				}
			}
			dom := mergeStrPtr(cur, func(row *orgbranding.Row) *string {
				if row == nil {
					return nil
				}
				return row.CustomDomain
			}, body.CustomDomain)
			email := mergeStrPtr(cur, func(row *orgbranding.Row) *string {
				if row == nil {
					return nil
				}
				return row.CustomEmailDisplayName
			}, body.CustomEmailDisplayName)
			if err := orgbranding.UpsertReplace(ctx, d.Pool, orgID, logo, fav, p1, p2, dom, email); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not update branding.")
				return
			}
			d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventOrgSettingsChange, "setting", &orgID, nil, raw)
			writeAdminConsoleSettings(w, ctx, d.Pool, orgID)
		default:
			w.Header().Set("Allow", "GET, PUT")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

func writeAdminConsoleSettings(w http.ResponseWriter, ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) {
	org, err := organization.GetByID(ctx, pool, orgID)
	if err != nil || org == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load organization.")
		return
	}
	branding, err := orgbranding.Get(ctx, pool, orgID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load branding.")
		return
	}
	primary := orgbranding.DefaultPrimaryHex
	secondary := orgbranding.DefaultSecondaryHex
	var logo, fav, dom, email *string
	if branding != nil {
		primary = branding.PrimaryColor
		secondary = branding.SecondaryColor
		logo = branding.LogoURL
		fav = branding.FaviconURL
		dom = branding.CustomDomain
		email = branding.CustomEmailDisplayName
	}
	var meta map[string]any
	_ = json.Unmarshal(org.Metadata, &meta)
	timezone, _ := meta["timezone"].(string)
	locale, _ := meta["locale"].(string)
	writeJSON(w, http.StatusOK, map[string]any{
		"orgId":                  orgID.String(),
		"name":                   org.Name,
		"slug":                   org.Slug,
		"logoUrl":                logo,
		"faviconUrl":             fav,
		"primaryColor":           primary,
		"secondaryColor":         secondary,
		"customDomain":           dom,
		"customEmailDisplayName": email,
		"timezone":               timezone,
		"locale":                 locale,
	})
}

func (d Deps) handleAdminConsoleDelegate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actor, orgID, ga, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		userID, role, _, _, parseErr := parseOrgRoleGrantPostBody(raw)
		if parseErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "user_id and role are required.")
			return
		}
		targetUID, err := uuid.Parse(userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
			return
		}
		if role == string(orgroles.RoleOrgAdmin) && !ga {
			can, err := orgrolegrant.CanManageOrgRoleGrants(r.Context(), d.Pool, actor, orgID, ga)
			if err != nil || !can {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Cannot delegate org_admin role.")
				return
			}
		}
		grant, err := orgroles.Create(r.Context(), d.Pool, orgID, targetUID, nil, orgroles.Role(role), &actor, nil)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not delegate role.")
			return
		}
		d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventRoleGrant, "user", &targetUID, nil, raw)
		writeJSON(w, http.StatusCreated, map[string]any{
			"id":     grant.ID.String(),
			"userId": grant.UserID.String(),
			"role":   string(grant.Role),
		})
	}
}

func (d Deps) handleAdminConsoleIntegrations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		cfg := d.effectiveConfig()
		ctx := r.Context()

		var webhookCount int64
		_ = d.Pool.QueryRow(ctx, `
SELECT COUNT(*)::bigint FROM integrations.webhook_subscriptions
WHERE org_id = $1 AND active = true
`, orgID).Scan(&webhookCount)

		var sisCount int64
		_ = d.Pool.QueryRow(ctx, `
SELECT COUNT(*)::bigint FROM sis.sis_connections WHERE org_id = $1
`, orgID).Scan(&sisCount)

		writeJSON(w, http.StatusOK, map[string]any{
			"orgId": orgID.String(),
			"sso": map[string]any{
				"saml":      cfg.SAMLSSOEnabled,
				"oidc":      cfg.OIDCSSOEnabled,
				"clever":    cfg.CleverSSOEnabled,
				"classlink": cfg.ClassLinkSSOEnabled,
			},
			"oneRoster": map[string]any{
				"enabled": cfg.OneRosterEnabled,
			},
			"scim": map[string]any{
				"enabled": cfg.ScimEnabled,
			},
			"sis": map[string]any{
				"enabled":           cfg.FFSISIntegration,
				"activeConnections": sisCount,
			},
			"webhooks": map[string]any{
				"enabled":       cfg.FFWebhooks,
				"subscriptions": webhookCount,
			},
		})
	}
}

func (d Deps) recordAdminConsoleAudit(r *http.Request, actor uuid.UUID, orgID *uuid.UUID, eventType string, targetType string, targetID *uuid.UUID, before, after []byte) {
	if !d.effectiveConfig().AdminAuditLogEnabled {
		return
	}
	tt := targetType
	ip := adminConsoleClientIP(r)
	ua := r.UserAgent()
	_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
		OrgID:       orgID,
		EventType:   eventType,
		ActorID:     actor,
		ActorIP:     &ip,
		UserAgent:   &ua,
		TargetType:  &tt,
		TargetID:    targetID,
		BeforeValue: before,
		AfterValue:  after,
	})
}

func adminConsoleClientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	return strings.TrimSpace(r.RemoteAddr)
}
