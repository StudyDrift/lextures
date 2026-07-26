package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) registerContentToolsStateRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/state", d.handleContentToolsStateGet())
	r.Put("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/state", d.handleContentToolsStatePut())
}

type contentToolsStatePutBody struct {
	StateJSON json.RawMessage `json:"stateJson"`
	Revision  int64           `json:"revision"`
}

func parseContentToolsStateScope(q string) (scope string, ok bool) {
	s := strings.TrimSpace(strings.ToLower(q))
	switch s {
	case "", ctrepo.ScopeEnrollment:
		return ctrepo.ScopeEnrollment, true
	case ctrepo.ScopePreview:
		return ctrepo.ScopePreview, true
	default:
		return "", false
	}
}

func (d Deps) handleContentToolsStateGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		scope, okScope := parseContentToolsStateScope(r.URL.Query().Get("scope"))
		if !okScope {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scope must be enrollment or preview.")
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load instance.")
			return
		}
		if inst == nil || inst.Status != "active" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}

		var enrollID *uuid.UUID
		if scope == ctrepo.ScopePreview {
			canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
				return
			}
			if !canEdit {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
				return
			}
			enrollID, err = enrollment.GetActiveEnrollmentID(r.Context(), d.Pool, courseID, viewer)
			if err != nil || enrollID == nil {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Active enrollment required.")
				return
			}
		} else {
			enrollID, err = enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, courseID, viewer)
			if err != nil || enrollID == nil {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Active enrollment required.")
				return
			}
		}

		var st *ctrepo.StateRow
		if scope == ctrepo.ScopePreview {
			st, err = ctrepo.GetStateByScope(r.Context(), d.Pool, instanceID, *enrollID, scope)
		} else {
			st, err = ctrepo.GetState(r.Context(), d.Pool, instanceID, *enrollID)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load state.")
			return
		}
		if st == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"stateJson": json.RawMessage(`{}`),
				"revision":  0,
				"status":    "not_started",
				"scope":     scope,
			})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stateJson": st.StateJSON,
			"revision":  st.Revision,
			"status":    st.Status,
			"scope":     st.Scope,
		})
	}
}

func (d Deps) handleContentToolsStatePut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		scope, okScope := parseContentToolsStateScope(r.URL.Query().Get("scope"))
		if !okScope {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scope must be enrollment or preview.")
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load instance.")
			return
		}
		if inst == nil || inst.Status != "active" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		m := ctsvc.MustDefault().Get(inst.ToolID)
		if m == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Tool no longer registered.")
			return
		}

		var enrollID *uuid.UUID
		if scope == ctrepo.ScopePreview {
			canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
				return
			}
			if !canEdit {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
				return
			}
			enrollID, err = enrollment.GetActiveEnrollmentID(r.Context(), d.Pool, courseID, viewer)
			if err != nil || enrollID == nil {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Active enrollment required.")
				return
			}
		} else {
			enrollID, err = enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, courseID, viewer)
			if err != nil || enrollID == nil {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Active enrollment required.")
				return
			}
		}

		var body contentToolsStatePutBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := ctsvc.ValidateStateJSON(m, body.StateJSON); err != nil {
			if err == ctsvc.ErrStateTooLarge {
				apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, err.Error())
				return
			}
			if ve, ok := err.(*ctsvc.ConfigValidationError); ok {
				writeContentToolsConfigValidation(w, ve)
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		var st *ctrepo.StateRow
		if scope == ctrepo.ScopePreview {
			st, err = ctrepo.UpsertPreviewState(r.Context(), d.Pool, instanceID, *enrollID, viewer, body.StateJSON, body.Revision)
		} else {
			st, err = ctrepo.UpsertState(r.Context(), d.Pool, instanceID, *enrollID, viewer, body.StateJSON, body.Revision)
		}
		if err != nil {
			if ctrepo.IsConfigSizeViolation(err) {
				apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, ctsvc.ErrStateTooLarge.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save state.")
			return
		}
		if st == nil {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Revision conflict.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"stateJson": st.StateJSON,
			"revision":  st.Revision,
			"status":    st.Status,
			"scope":     st.Scope,
		})
	}
}
