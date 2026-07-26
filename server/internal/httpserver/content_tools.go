package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/course"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) registerContentToolsRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/content-tools/catalog", d.handleContentToolsCatalog())
	r.Get("/api/v1/courses/{course_code}/content-tools/manifests/{tool_id}", d.handleContentToolsManifestGet())
	r.Get("/api/v1/courses/{course_code}/content-tools/settings", d.handleContentToolsSettingsGet())
	r.Put("/api/v1/courses/{course_code}/content-tools/settings", d.handleContentToolsSettingsPut())
	r.Get("/api/v1/courses/{course_code}/content-tools/instances", d.handleContentToolsInstancesList())
	r.Post("/api/v1/courses/{course_code}/content-tools/instances", d.handleContentToolsInstancesCreate())
	r.Patch("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}", d.handleContentToolsInstancePatch())
	r.Delete("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}", d.handleContentToolsInstanceDelete())
	d.registerContentToolsAuthoringRoutes(r)
	d.registerContentToolsStateRoutes(r)
}

func writeContentToolsUnavailable(w http.ResponseWriter) {
	ctsvc.RefreshKillSwitchMetric()
	apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
}

func writeContentToolsConfigValidation(w http.ResponseWriter, err *ctsvc.ConfigValidationError) {
	errs := make([]ctmodel.FieldError, 0, len(err.Errors))
	for _, e := range err.Errors {
		errs = append(errs, ctmodel.FieldError{Path: e.Path, Message: e.Message})
	}
	var body ctmodel.ValidationErrorBody
	body.Error.Code = apierr.CodeUnprocessableEntity
	body.Error.Message = "Config validation failed."
	body.Error.Errors = errs
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(body)
}

// requireContentToolsCourse loads the course, enforces membership, and ensures
// Content Tools is available (flag on + kill-switch off). Otherwise 404 (FR-14).
func (d Deps) requireContentToolsCourse(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, courseID uuid.UUID, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	pub, err := course.GetPublicByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if pub == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if !ctsvc.AvailableForCourse(pub.ContentToolsEnabled) {
		writeContentToolsUnavailable(w)
		return "", uuid.Nil, uuid.Nil, false
	}
	cid, err := uuid.Parse(pub.ID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid course id.")
		return "", uuid.Nil, uuid.Nil, false
	}
	return courseCode, viewer, cid, true
}

func (d Deps) viewerCanEditContentTools(ctx context.Context, courseCode string, viewer uuid.UUID) (bool, error) {
	return courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":item:create")
}

func contentToolsSettingsToAPI(r ctrepo.SettingsRow) ctmodel.Settings {
	ids := r.AllowedToolIDs
	if ids == nil {
		ids = []string{}
	}
	maxInst := r.MaxInstancesPerItem
	if maxInst <= 0 {
		maxInst = 50
	}
	return ctmodel.Settings{
		AllowedToolIDs:      ids,
		StudentResetAllowed: r.StudentResetAllowed,
		MaxInstancesPerItem: maxInst,
		UpdatedAt:           r.UpdatedAt,
	}
}

func instanceToAPI(row ctrepo.InstanceRow, config json.RawMessage) ctmodel.ToolInstance {
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	return ctmodel.ToolInstance{
		ID:              row.ID,
		ToolID:          row.ToolID,
		ToolVersion:     row.ToolVersion,
		HostKind:        row.HostKind,
		StructureItemID: row.StructureItemID,
		SectionKey:      row.SectionKey,
		Title:           row.Title,
		Config:          config,
		Status:          row.Status,
		UpdatedAt:       row.UpdatedAt,
	}
}

func (d Deps) handleContentToolsCatalog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		settings, err := ctrepo.GetSettings(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
			return
		}
		allowed := []string{}
		if settings != nil {
			allowed = settings.AllowedToolIDs
		}
		reg := ctsvc.MustDefault()
		tools := ctsvc.FilterCatalog(reg, allowed, "", nil)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"tools": tools})
	}
}

func (d Deps) handleContentToolsManifestGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _, _, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		toolID := chi.URLParam(r, "tool_id")
		m := ctsvc.MustDefault().Get(toolID)
		if m == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tool not found.")
			return
		}
		pub, err := ctsvc.ManifestToPublic(m)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load manifest.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(pub)
	}
}

func (d Deps) handleContentToolsSettingsGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		row, err := ctrepo.GetSettings(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
			return
		}
		if row == nil {
			def := ctrepo.DefaultSettings(courseID)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(contentToolsSettingsToAPI(def))
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(contentToolsSettingsToAPI(*row))
	}
}

func (d Deps) handleContentToolsSettingsPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var body ctmodel.Settings
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.MaxInstancesPerItem == 0 {
			body.MaxInstancesPerItem = 50
		}
		if body.MaxInstancesPerItem < 1 || body.MaxInstancesPerItem > 200 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "maxInstancesPerItem must be between 1 and 200.")
			return
		}
		reg := ctsvc.MustDefault()
		for _, id := range body.AllowedToolIDs {
			if reg.Get(id) == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown tool id in allowlist: "+id)
				return
			}
		}
		if body.AllowedToolIDs == nil {
			body.AllowedToolIDs = []string{}
		}
		row, err := ctrepo.UpsertSettings(r.Context(), d.Pool, courseID, ctrepo.SettingsRow{
			CourseID:            courseID,
			AllowedToolIDs:      body.AllowedToolIDs,
			StudentResetAllowed: body.StudentResetAllowed,
			MaxInstancesPerItem: body.MaxInstancesPerItem,
		}, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save settings.")
			return
		}
		actor := viewer
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, nil, nil, &actor, "_settings", ctsvc.EventSettingsUpdated, map[string]any{
			"allowedToolIds":      body.AllowedToolIDs,
			"studentResetAllowed": body.StudentResetAllowed,
			"maxInstancesPerItem": body.MaxInstancesPerItem,
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(contentToolsSettingsToAPI(*row))
	}
}

func (d Deps) handleContentToolsInstancesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		var itemID *uuid.UUID
		if q := r.URL.Query().Get("itemId"); q != "" {
			parsed, err := uuid.Parse(q)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid itemId.")
				return
			}
			itemID = &parsed
		}
		hostKind := r.URL.Query().Get("hostKind")
		rows, err := ctrepo.ListInstances(r.Context(), d.Pool, courseID, itemID, hostKind, false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list instances.")
			return
		}
		reg := ctsvc.MustDefault()
		out := make([]ctmodel.ToolInstance, 0, len(rows))
		for _, row := range rows {
			cfg := row.ConfigJSON
			if !canEdit {
				m := reg.Get(row.ToolID)
				if m != nil {
					redacted, rerr := ctsvc.RedactSensitiveConfig(m.ConfigSchema, cfg)
					if rerr == nil {
						cfg = redacted
					}
				}
			}
			out = append(out, instanceToAPI(row, cfg))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"instances": out})
	}
}

func (d Deps) handleContentToolsInstancesCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		var body ctmodel.CreateInstanceRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := ctsvc.ValidateHostKindShape(body.HostKind, body.StructureItemID != nil); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		reg := ctsvc.MustDefault()
		m := reg.Get(body.ToolID)
		if m == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown toolId.")
			return
		}
		settings, err := ctrepo.GetSettings(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
			return
		}
		allowed := []string{}
		maxInst := int16(50)
		if settings != nil {
			allowed = settings.AllowedToolIDs
			maxInst = settings.MaxInstancesPerItem
		}
		if !ctsvc.ToolAllowedByAllowlist(allowed, body.ToolID) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, ctsvc.ErrToolNotAllowed.Error())
			return
		}
		if body.StructureItemID != nil {
			inCourse, err := ctrepo.StructureItemInCourse(r.Context(), d.Pool, courseID, *body.StructureItemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify structure item.")
				return
			}
			if !inCourse {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, ctsvc.ErrItemNotInCourse.Error())
				return
			}
		}
		n, err := ctrepo.CountActiveForItem(r.Context(), d.Pool, courseID, body.StructureItemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to count instances.")
			return
		}
		if int16(n) >= maxInst {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, ctsvc.ErrMaxInstances.Error())
			return
		}
		if err := ctsvc.ValidateConfigJSON(m, body.Config); err != nil {
			if ve, ok := err.(*ctsvc.ConfigValidationError); ok {
				ctsvc.IncConfigValidationFailure(body.ToolID)
				writeContentToolsConfigValidation(w, ve)
				return
			}
			if err == ctsvc.ErrConfigTooLarge {
				apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		actor := viewer
		created, err := ctrepo.CreateInstance(r.Context(), d.Pool, ctrepo.InstanceRow{
			CourseID:        courseID,
			StructureItemID: body.StructureItemID,
			HostKind:        body.HostKind,
			SectionKey:      body.SectionKey,
			ToolID:          body.ToolID,
			ToolVersion:     m.Version,
			Title:           body.Title,
			ConfigJSON:      body.Config,
			CreatedBy:       &actor,
		})
		if err != nil {
			if ctrepo.IsConfigSizeViolation(err) {
				apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, ctsvc.ErrConfigTooLarge.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create instance.")
			return
		}
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &created.ID, nil, &actor, created.ToolID, ctsvc.EventInstanceCreated, map[string]any{
			"hostKind": created.HostKind,
		})
		ctsvc.IncInstanceAction(created.ToolID, "create")
		ctsvc.IncInsert(created.ToolID, "api")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(instanceToAPI(*created, created.ConfigJSON))
	}
}

func (d Deps) handleContentToolsInstancePatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		existing, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load instance.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		var body ctmodel.PatchInstanceRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Status != nil {
			if _, ok := map[string]struct{}{"active": {}, "archived": {}}[*body.Status]; !ok {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, ctsvc.ErrInvalidStatus.Error())
				return
			}
		}
		var cfg json.RawMessage
		if body.Config != nil {
			m := ctsvc.MustDefault().Get(existing.ToolID)
			if m == nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Tool no longer registered.")
				return
			}
			if err := ctsvc.ValidateConfigJSON(m, *body.Config); err != nil {
				if ve, ok := err.(*ctsvc.ConfigValidationError); ok {
					ctsvc.IncConfigValidationFailure(existing.ToolID)
					ctsvc.IncConfigSave(existing.ToolID, "validation_error")
					writeContentToolsConfigValidation(w, ve)
					return
				}
				if err == ctsvc.ErrConfigTooLarge {
					ctsvc.IncConfigSave(existing.ToolID, "too_large")
					apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, err.Error())
					return
				}
				ctsvc.IncConfigSave(existing.ToolID, "error")
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			cfg = *body.Config
		}
		updated, err := ctrepo.UpdateInstance(r.Context(), d.Pool, courseID, instanceID, body.Title, body.SectionKey, cfg, body.Status)
		if err != nil {
			if ctrepo.IsConfigSizeViolation(err) {
				ctsvc.IncConfigSave(existing.ToolID, "too_large")
				apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, ctsvc.ErrConfigTooLarge.Error())
				return
			}
			ctsvc.IncConfigSave(existing.ToolID, "error")
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update instance.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		actor := viewer
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &updated.ID, nil, &actor, updated.ToolID, ctsvc.EventInstanceUpdated, map[string]any{})
		ctsvc.IncInstanceAction(updated.ToolID, "update")
		if body.Config != nil {
			ctsvc.IncConfigSave(updated.ToolID, "ok")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(instanceToAPI(*updated, updated.ConfigJSON))
	}
}

func (d Deps) handleContentToolsInstanceDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		permanent := strings.EqualFold(r.URL.Query().Get("permanent"), "true")
		actor := viewer
		if permanent {
			existing, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load instance.")
				return
			}
			if existing == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
				return
			}
			if err := ctrepo.HardDeleteInstance(r.Context(), d.Pool, courseID, instanceID); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete instance.")
				return
			}
			// instance_id is CASCADE-null-unsafe; record course-scoped audit with id in payload.
			_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, nil, nil, &actor, existing.ToolID, ctsvc.EventInstanceDeleted, map[string]any{
				"permanent":  true,
				"instanceId": existing.ID.String(),
			})
			ctsvc.IncInstanceAction(existing.ToolID, "delete")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		archived, err := ctrepo.ArchiveInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to archive instance.")
			return
		}
		if archived == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &archived.ID, nil, &actor, archived.ToolID, ctsvc.EventInstanceArchived, map[string]any{})
		ctsvc.IncInstanceAction(archived.ToolID, "archive")
		w.WriteHeader(http.StatusNoContent)
	}
}
