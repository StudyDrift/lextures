package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

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
		withState := strings.TrimSpace(r.URL.Query().Get("withState")) == "1"
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

		if withState && len(out) > 0 {
			enrollID, err := enrollment.GetActiveEnrollmentID(r.Context(), d.Pool, courseID, viewer)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve enrollment.")
				return
			}
			if enrollID != nil {
				ids := make([]uuid.UUID, 0, len(out))
				for _, inst := range out {
					ids = append(ids, inst.ID)
				}
				states, err := ctrepo.ListStatesForEnrollment(r.Context(), d.Pool, *enrollID, ids)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load states.")
					return
				}
				for i := range out {
					if st, ok := states[out[i].ID]; ok && st != nil {
						env := d.contentToolsStateEnvelopeMigrated(r.Context(), out[i].ToolID, out[i].ID, st)
						out[i].State = &env
					} else {
						env := contentToolsEmptyEnvelope(out[i].ID, ctrepo.ScopeEnrollment)
						out[i].State = &env
					}
				}
			}
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
		// CT.5: apply any registered config migrations from v1 toward current.
		if table := ctsvc.DefaultMigrations().Get(body.ToolID); table != nil {
			res := ctsvc.ApplyConfigMigrations(table, 1, body.Config)
			if !res.Quarantine && !res.Unchanged {
				body.Config = res.Doc
			}
			_ = ctsvc.MigrationFromVersions(table.Config)
		}
		actor := viewer
		created, err := ctrepo.CreateInstance(r.Context(), d.Pool, ctrepo.InstanceRow{
			CourseID:        courseID,
			StructureItemID: body.StructureItemID,
			HostKind:        body.HostKind,
			SectionKey:      body.SectionKey,
			ToolID:          body.ToolID,
			ToolVersion:     resolveContentToolVersion(r.Context(), d.Pool, m),
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
