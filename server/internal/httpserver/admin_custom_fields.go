package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	cfrepo "github.com/lextures/lextures/server/internal/repos/customfields"
	cfsvc "github.com/lextures/lextures/server/internal/service/customfields"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
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

func (d Deps) customFieldsService() cfsvc.Service {
	return cfsvc.Service{Pool: d.Pool}
}

func (d Deps) registerAdminCustomFieldRoutes(r chi.Router) {
	r.Get("/api/v1/admin-console/custom-fields", d.handleAdminCustomFieldsList())
	r.Post("/api/v1/admin-console/custom-fields", d.handleAdminCustomFieldsCreate())
	r.Put("/api/v1/admin-console/custom-fields/reorder", d.handleAdminCustomFieldsReorder())
	r.Put("/api/v1/admin-console/custom-fields/{fieldId}", d.handleAdminCustomFieldsUpdate())
	r.Delete("/api/v1/admin-console/custom-fields/{fieldId}", d.handleAdminCustomFieldsDelete())
	r.Get("/api/v1/admin-console/users/export", d.handleAdminConsoleUsersExport())
	r.Get("/api/v1/admin-console/users/{userId}", d.handleAdminConsoleUserGet())
}

type customFieldDefinitionJSON struct {
	ID            string   `json:"id"`
	OrgID         string   `json:"orgId"`
	EntityType    string   `json:"entityType"`
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	FieldType     string   `json:"fieldType"`
	SelectOptions []string `json:"selectOptions,omitempty"`
	IsRequired    bool     `json:"isRequired"`
	Visibility    string   `json:"visibility"`
	SortOrder     int      `json:"sortOrder"`
	CreatedAt     string   `json:"createdAt"`
}

func toCustomFieldDefinitionJSON(d cfrepo.Definition) customFieldDefinitionJSON {
	opts := d.SelectOptions
	if opts == nil {
		opts = []string{}
	}
	return customFieldDefinitionJSON{
		ID: d.ID.String(), OrgID: d.OrgID.String(), EntityType: string(d.EntityType),
		Key: d.Key, Label: d.Label, FieldType: string(d.FieldType), SelectOptions: opts,
		IsRequired: d.IsRequired, Visibility: string(d.Visibility), SortOrder: d.SortOrder,
		CreatedAt: d.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func queryIncludes(r *http.Request, name string) bool {
	for _, part := range strings.Split(r.URL.Query().Get("include"), ",") {
		if strings.TrimSpace(part) == name {
			return true
		}
	}
	return false
}

func customfieldsValidVisibility(v cfrepo.Visibility) bool {
	switch v {
	case cfrepo.VisibilityAdminOnly, cfrepo.VisibilityInstructor, cfrepo.VisibilityStudent:
		return true
	default:
		return false
	}
}

func customfieldsFormatExportValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func (d Deps) recordCustomFieldAudit(r *http.Request, actor, orgID uuid.UUID, eventType string, targetID uuid.UUID, before, after []byte) {
	targetType := "custom_field_definition"
	d.recordAdminConsoleAudit(r, actor, &orgID, eventType, targetType, &targetID, before, after)
}

func (d Deps) handleAdminCustomFieldsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.customFieldsEnabled(w) {
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		entityType, err := cfsvc.ParseEntityType(r.URL.Query().Get("entity_type"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		defs, err := cfrepo.ListDefinitions(r.Context(), d.Pool, orgID, entityType)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list custom fields.")
			return
		}
		out := make([]customFieldDefinitionJSON, 0, len(defs))
		for _, def := range defs {
			out = append(out, toCustomFieldDefinitionJSON(def))
		}
		writeJSON(w, http.StatusOK, out)
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
			SortOrder     *int     `json:"sortOrder"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		entityType, err := cfsvc.ParseEntityType(body.EntityType)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		fieldType := cfrepo.FieldType(strings.TrimSpace(body.FieldType))
		visibility := cfrepo.Visibility(strings.TrimSpace(body.Visibility))
		if visibility == "" {
			visibility = cfrepo.VisibilityAdminOnly
		}
		if errs := cfsvc.ValidateDefinitionInput(body.Key, body.Label, fieldType, body.SelectOptions); len(errs) > 0 {
			apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, errs.Error())
			return
		}
		if !customfieldsValidVisibility(visibility) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid visibility.")
			return
		}
		sortOrder := 0
		if body.SortOrder != nil {
			sortOrder = *body.SortOrder
		}
		def, err := cfrepo.CreateDefinition(r.Context(), d.Pool, cfrepo.CreateParams{
			OrgID: orgID, EntityType: entityType, Key: strings.TrimSpace(body.Key),
			Label: strings.TrimSpace(body.Label), FieldType: fieldType,
			SelectOptions: body.SelectOptions, IsRequired: body.IsRequired,
			Visibility: visibility, SortOrder: sortOrder,
		})
		if err == cfrepo.ErrDuplicateKey {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "A custom field with this key already exists.")
			return
		}
		if err == cfrepo.ErrMaxFields {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Maximum custom fields per entity type reached.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create custom field.")
			return
		}
		d.recordCustomFieldAudit(r, actor, orgID, auditservice.EventCustomFieldDefinitionChange, def.ID, nil, raw)
		writeJSON(w, http.StatusCreated, toCustomFieldDefinitionJSON(def))
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
		before, _ := cfrepo.GetDefinition(r.Context(), d.Pool, orgID, fieldID)
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body struct {
			Label         *string   `json:"label"`
			FieldType     *string   `json:"fieldType"`
			SelectOptions *[]string `json:"selectOptions"`
			IsRequired    *bool     `json:"isRequired"`
			Visibility    *string   `json:"visibility"`
			SortOrder     *int      `json:"sortOrder"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var fieldType *cfrepo.FieldType
		if body.FieldType != nil {
			ft := cfrepo.FieldType(strings.TrimSpace(*body.FieldType))
			fieldType = &ft
		}
		var visibility *cfrepo.Visibility
		if body.Visibility != nil {
			v := cfrepo.Visibility(strings.TrimSpace(*body.Visibility))
			if !customfieldsValidVisibility(v) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid visibility.")
				return
			}
			visibility = &v
		}
		def, err := cfrepo.UpdateDefinition(r.Context(), d.Pool, orgID, fieldID, cfrepo.UpdateParams{
			Label: body.Label, FieldType: fieldType, SelectOptions: body.SelectOptions,
			IsRequired: body.IsRequired, Visibility: visibility, SortOrder: body.SortOrder,
		})
		if err == cfrepo.ErrNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Custom field not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update custom field.")
			return
		}
		var beforeRaw []byte
		if before.ID != uuid.Nil {
			beforeRaw, _ = json.Marshal(toCustomFieldDefinitionJSON(before))
		}
		d.recordCustomFieldAudit(r, actor, orgID, auditservice.EventCustomFieldDefinitionChange, def.ID, beforeRaw, raw)
		writeJSON(w, http.StatusOK, toCustomFieldDefinitionJSON(def))
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
		before, _ := cfrepo.GetDefinition(r.Context(), d.Pool, orgID, fieldID)
		if err := cfrepo.SoftDeleteDefinition(r.Context(), d.Pool, orgID, fieldID); err == cfrepo.ErrNotFound {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Custom field not found.")
			return
		} else if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete custom field.")
			return
		}
		var beforeRaw []byte
		if before.ID != uuid.Nil {
			beforeRaw, _ = json.Marshal(toCustomFieldDefinitionJSON(before))
		}
		d.recordCustomFieldAudit(r, actor, orgID, auditservice.EventCustomFieldDefinitionChange, fieldID, beforeRaw, nil)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleAdminCustomFieldsReorder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.customFieldsEnabled(w) {
			return
		}
		_, orgID, _, ok := d.adminConsoleAccess(w, r, true)
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
		entityType, err := cfsvc.ParseEntityType(body.EntityType)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
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
		if err := cfrepo.ReorderDefinitions(r.Context(), d.Pool, orgID, entityType, ids); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		defs, err := cfrepo.ListDefinitions(r.Context(), d.Pool, orgID, entityType)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list custom fields.")
			return
		}
		out := make([]customFieldDefinitionJSON, 0, len(defs))
		for _, def := range defs {
			out = append(out, toCustomFieldDefinitionJSON(def))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (d Deps) handleAdminConsoleUserGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, orgID, _, ok := d.adminConsoleAccess(w, r, false)
		if !ok {
			return
		}
		targetID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "userId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid user id.")
			return
		}
		ctx := r.Context()
		var targetOrg uuid.UUID
		var email, firstName, lastName string
		var displayName *string
		var active bool
		var createdAt string
		err = d.Pool.QueryRow(ctx, `
SELECT org_id, email, COALESCE(first_name,''), COALESCE(last_name,''), display_name,
       (deactivated_at IS NULL AND NOT login_blocked), created_at::text
FROM "user".users WHERE id = $1
`, targetID).Scan(&targetOrg, &email, &firstName, &lastName, &displayName, &active, &createdAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		if targetOrg != orgID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		out := map[string]any{
			"id": targetID.String(), "email": email, "displayName": displayName,
			"firstName": firstName, "lastName": lastName, "active": active, "createdAt": createdAt,
		}
		if queryIncludes(r, "custom_fields") && d.effectiveConfig().CustomFieldsEnabled {
			fields, err := d.customFieldsService().UserCustomFieldsForViewer(ctx, orgID, targetID, cfsvc.ViewerAdmin)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load custom fields.")
				return
			}
			out["customFields"] = fields
		}
		writeJSON(w, http.StatusOK, out)
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
		defs, err := cfrepo.ListDefinitions(ctx, d.Pool, orgID, cfrepo.EntityUser)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load custom fields.")
			return
		}
		rows, err := d.Pool.Query(ctx, `
SELECT email, COALESCE(first_name,''), COALESCE(last_name,''), custom_fields
FROM "user".users
WHERE org_id = $1 AND id <> 'a0000000-0000-4000-8000-000000000001'::uuid
ORDER BY email
`, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export users.")
			return
		}
		defer rows.Close()

		var buf bytes.Buffer
		header := []string{"email", "first_name", "last_name"}
		for _, def := range defs {
			header = append(header, def.Label)
		}
		buf.WriteString(strings.Join(header, ",") + "\n")

		for rows.Next() {
			var email, firstName, lastName string
			var rawFields []byte
			if err := rows.Scan(&email, &firstName, &lastName, &rawFields); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export users.")
				return
			}
			fields := map[string]any{}
			if len(rawFields) > 0 {
				_ = json.Unmarshal(rawFields, &fields)
			}
			line := []string{csvEscapeField(email), csvEscapeField(firstName), csvEscapeField(lastName)}
			for _, def := range defs {
				val := ""
				if v, ok := fields[def.Key]; ok && v != nil {
					val = customfieldsFormatExportValue(v)
				}
				line = append(line, csvEscapeField(val))
			}
			buf.WriteString(strings.Join(line, ",") + "\n")
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="users-export.csv"`)
		_, _ = w.Write(buf.Bytes())
	}
}

func csvEscapeField(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
