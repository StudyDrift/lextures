package httpserver

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/apierr"
	cfrepo "github.com/lextures/lextures/server/internal/repos/customfields"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	cfservice "github.com/lextures/lextures/server/internal/service/customfields"
)

func (d Deps) customFieldsEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().CustomFieldsEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Custom fields are not enabled.")
		return false
	}
	if !d.adminConsoleEnabled(w) {
		return false
	}
	return true
}

func (d Deps) registerAdminCustomFieldRoutes(r chi.Router) {
	r.Get("/api/v1/admin-console/custom-fields", d.handleAdminCustomFieldsList())
	r.Post("/api/v1/admin-console/custom-fields", d.handleAdminCustomFieldsCreate())
	r.Put("/api/v1/admin-console/custom-fields/{fieldId}", d.handleAdminCustomFieldsUpdate())
	r.Delete("/api/v1/admin-console/custom-fields/{fieldId}", d.handleAdminCustomFieldsDelete())
	r.Put("/api/v1/admin-console/custom-fields/reorder", d.handleAdminCustomFieldsReorder())
	r.Patch("/api/v1/admin-console/courses/{courseId}", d.handleAdminConsoleCoursePatch())
	r.Patch("/api/v1/admin-console/enrollments/{enrollmentId}", d.handleAdminConsoleEnrollmentPatch())
}

func (d Deps) customFieldsService() *cfservice.Service {
	return cfservice.New(d.Pool)
}

func parseEntityType(raw string) (cfrepo.EntityType, bool) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "user":
		return cfrepo.EntityUser, true
	case "course":
		return cfrepo.EntityCourse, true
	case "enrollment":
		return cfrepo.EntityEnrollment, true
	default:
		return "", false
	}
}

func definitionToJSON(d cfrepo.Definition) map[string]any {
	out := map[string]any{
		"id":          d.ID.String(),
		"orgId":       d.OrgID.String(),
		"entityType":  string(d.EntityType),
		"key":         d.Key,
		"label":       d.Label,
		"fieldType":   string(d.FieldType),
		"isRequired":  d.IsRequired,
		"visibility":  string(d.Visibility),
		"sortOrder":   d.SortOrder,
		"createdAt":   d.CreatedAt,
	}
	if len(d.SelectOptions) > 0 {
		out["selectOptions"] = d.SelectOptions
	}
	if d.DeletedAt != nil {
		out["deletedAt"] = d.DeletedAt
	}
	return out
}

func writeCustomFieldValidationErrors(w http.ResponseWriter, errs []cfservice.ValidationError) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error": map[string]any{
			"code":    apierr.CodeUnprocessableEntity,
			"message": "Custom field validation failed.",
		},
		"errors": errs,
	})
}

func (d Deps) handleAdminCustomFieldsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.customFieldsEnabled(w) {
			return
		}
		_, orgID, globalAdmin, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		entityType, okType := parseEntityType(r.URL.Query().Get("entity_type"))
		if !okType {
			entityType, okType = parseEntityType(r.URL.Query().Get("entityType"))
		}
		if !okType {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "entity_type is required (user, course, or enrollment).")
			return
		}
		includeDeleted := globalAdmin && strings.EqualFold(r.URL.Query().Get("include_deleted"), "true")
		defs, err := d.customFieldsService().ListDefinitions(r.Context(), orgID, entityType, includeDeleted)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list custom fields.")
			return
		}
		items := make([]map[string]any, 0, len(defs))
		for _, def := range defs {
			items = append(items, definitionToJSON(def))
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func (d Deps) handleAdminCustomFieldsCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.customFieldsEnabled(w) {
			return
		}
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			EntityType    string   `json:"entityType"`
			Key           string   `json:"key"`
			Label         string   `json:"label"`
			FieldType     string   `json:"fieldType"`
			SelectOptions []string `json:"selectOptions"`
			IsRequired    bool     `json:"isRequired"`
			Visibility    string   `json:"visibility"`
			SortOrder     int      `json:"sortOrder"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		entityType, okType := parseEntityType(body.EntityType)
		if !okType {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid entityType.")
			return
		}
		def, err := d.customFieldsService().CreateDefinition(
			r.Context(), orgID, entityType, body.Key, body.Label,
			cfrepo.FieldType(body.FieldType), body.SelectOptions, body.IsRequired,
			cfrepo.Visibility(body.Visibility), body.SortOrder,
		)
		if err != nil {
			switch err {
			case cfrepo.ErrDuplicateKey:
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "A custom field with this key already exists.")
			case cfrepo.ErrMaxFields:
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity, "Maximum custom fields per entity type reached.")
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			}
			return
		}
		d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventCustomFieldDefinitionChange, "custom_field_definition", &def.ID, nil, raw)
		writeJSON(w, http.StatusCreated, definitionToJSON(*def))
	}
}

func (d Deps) handleAdminCustomFieldsUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.customFieldsEnabled(w) {
			return
		}
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		fieldID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "fieldId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid field id.")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			Label         *string  `json:"label"`
			SelectOptions []string `json:"selectOptions"`
			IsRequired    *bool    `json:"isRequired"`
			Visibility    *string  `json:"visibility"`
			SortOrder     *int     `json:"sortOrder"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var visibility *cfrepo.Visibility
		if body.Visibility != nil {
			v := cfrepo.Visibility(*body.Visibility)
			visibility = &v
		}
		var selectOpts []string
		if body.SelectOptions != nil {
			selectOpts = body.SelectOptions
		}
		def, err := d.customFieldsService().UpdateDefinition(r.Context(), orgID, fieldID, body.Label, selectOpts, body.IsRequired, visibility, body.SortOrder)
		if err == cfrepo.ErrNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Custom field not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventCustomFieldDefinitionChange, "custom_field_definition", &fieldID, nil, raw)
		writeJSON(w, http.StatusOK, definitionToJSON(*def))
	}
}

func (d Deps) handleAdminCustomFieldsDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.customFieldsEnabled(w) {
			return
		}
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		fieldID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "fieldId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid field id.")
			return
		}
		if err := d.customFieldsService().SoftDeleteDefinition(r.Context(), orgID, fieldID); err == cfrepo.ErrNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Custom field not found.")
			return
		} else if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete custom field.")
			return
		}
		d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventCustomFieldDefinitionChange, "custom_field_definition", &fieldID, nil, []byte(`{"deleted":true}`))
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleAdminCustomFieldsReorder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.customFieldsEnabled(w) {
			return
		}
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			EntityType string   `json:"entityType"`
			FieldIDs   []string `json:"fieldIds"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		entityType, okType := parseEntityType(body.EntityType)
		if !okType {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid entityType.")
			return
		}
		ids := make([]uuid.UUID, 0, len(body.FieldIDs))
		for _, s := range body.FieldIDs {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid field id in fieldIds.")
				return
			}
			ids = append(ids, id)
		}
		if err := d.customFieldsService().ReorderDefinitions(r.Context(), orgID, entityType, ids); err == cfrepo.ErrNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Custom field not found.")
			return
		} else if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reorder custom fields.")
			return
		}
		d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventCustomFieldDefinitionChange, "custom_field_definition", nil, nil, raw)
		defs, err := d.customFieldsService().ListDefinitions(r.Context(), orgID, entityType, false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list custom fields.")
			return
		}
		items := make([]map[string]any, 0, len(defs))
		for _, def := range defs {
			items = append(items, definitionToJSON(def))
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func (d Deps) handleAdminConsoleUserGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.adminConsoleEnabled(w) {
			return
		}
		_, orgID, globalAdmin, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		userID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "userId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
			return
		}
		result, err := adminConsoleUserByID(r.Context(), d.Pool, orgID, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		if d.effectiveConfig().CustomFieldsEnabled && wantsInclude(r, "custom_fields") {
			audience := cfservice.AudienceAdmin
			includeDeleted := globalAdmin
			fields, err := d.customFieldsService().GetUserValues(r.Context(), orgID, userID, audience, includeDeleted)
			if err == nil {
				result["customFields"] = fields
			}
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (d Deps) handleAdminConsoleUsersExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.customFieldsEnabled(w) {
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		ctx := r.Context()
		defs, err := d.customFieldsService().ListDefinitions(ctx, orgID, cfrepo.EntityUser, false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load custom fields.")
			return
		}
		rows, err := d.Pool.Query(ctx, `
SELECT u.id::text, u.email, COALESCE(u.first_name,''), COALESCE(u.last_name,''),
       COALESCE((SELECT ar.name FROM "user".user_app_roles uar
        JOIN "user".app_roles ar ON ar.id = uar.role_id
        WHERE uar.user_id = u.id ORDER BY ar.name LIMIT 1), '') AS role,
       COALESCE(u.external_id,''), u.custom_fields
FROM "user".users u
WHERE u.org_id = $1 AND u.id <> 'a0000000-0000-4000-8000-000000000001'::uuid
ORDER BY u.email
`, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export users.")
			return
		}
		defer rows.Close()

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="users-export.csv"`)
		cw := csv.NewWriter(w)
		header := []string{"email", "first_name", "last_name", "role", "external_id"}
		for _, def := range defs {
			header = append(header, def.Label)
		}
		_ = cw.Write(header)
		for rows.Next() {
			var id, email, first, last, role, extID string
			var customRaw []byte
			if err := rows.Scan(&id, &email, &first, &last, &role, &extID, &customRaw); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export users.")
				return
			}
			var custom map[string]any
			_ = json.Unmarshal(customRaw, &custom)
			rec := []string{email, first, last, strings.ToLower(role), extID}
			for _, def := range defs {
				rec = append(rec, formatCSVValue(custom[def.Key]))
			}
			_ = cw.Write(rec)
		}
		cw.Flush()
	}
}

func (d Deps) handleAdminConsoleCoursePatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.customFieldsEnabled(w) {
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
			CustomFields map[string]any `json:"customFields"`
		}
		if err := json.Unmarshal(raw, &body); err != nil || body.CustomFields == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "customFields object is required.")
			return
		}
		merged, valErrs, err := d.customFieldsService().SetCourseValues(r.Context(), orgID, courseID, body.CustomFields)
		if err == cfrepo.ErrNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update course custom fields.")
			return
		}
		if len(valErrs) > 0 {
			writeCustomFieldValidationErrors(w, valErrs)
			return
		}
		d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventOrgSettingsChange, "course", &courseID, nil, raw)
		writeJSON(w, http.StatusOK, map[string]any{"id": courseID.String(), "customFields": merged})
	}
}

func (d Deps) handleAdminConsoleEnrollmentPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.customFieldsEnabled(w) {
			return
		}
		actor, orgID, _, ok := d.adminConsoleAccess(w, r, true)
		if !ok {
			return
		}
		enrollmentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "enrollmentId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			CustomFields map[string]any `json:"customFields"`
		}
		if err := json.Unmarshal(raw, &body); err != nil || body.CustomFields == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "customFields object is required.")
			return
		}
		merged, valErrs, err := d.customFieldsService().SetEnrollmentValues(r.Context(), orgID, enrollmentID, body.CustomFields)
		if err == cfrepo.ErrNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update enrollment custom fields.")
			return
		}
		if len(valErrs) > 0 {
			writeCustomFieldValidationErrors(w, valErrs)
			return
		}
		d.recordAdminConsoleAudit(r, actor, &orgID, auditservice.EventEnrollmentCreate, "enrollment", &enrollmentID, nil, raw)
		writeJSON(w, http.StatusOK, map[string]any{"id": enrollmentID.String(), "customFields": merged})
	}
}

func formatCSVValue(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func adminConsoleUserByID(ctx context.Context, pool *pgxpool.Pool, orgID, userID uuid.UUID) (map[string]any, error) {
	var id, email, role string
	var displayName *string
	var active bool
	var createdAt any
	err := pool.QueryRow(ctx, `
SELECT u.id::text, u.email, u.display_name,
       COALESCE((SELECT ar.name FROM "user".user_app_roles uar
        JOIN "user".app_roles ar ON ar.id = uar.role_id
        WHERE uar.user_id = u.id ORDER BY ar.name LIMIT 1), '') AS role,
       (u.deactivated_at IS NULL AND NOT u.login_blocked) AS active,
       u.created_at
FROM "user".users u
WHERE u.id = $1 AND u.org_id = $2
`, userID, orgID).Scan(&id, &email, &displayName, &role, &active, &createdAt)
	if err == pgx.ErrNoRows {
		return nil, pgx.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":          id,
		"email":       email,
		"displayName": displayName,
		"role":        strings.ToLower(role),
		"active":      active,
		"createdAt":   createdAt,
	}, nil
}
