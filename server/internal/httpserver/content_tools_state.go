package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/config"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/ratelimit"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) registerContentToolsStateRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/state", d.handleContentToolsStateGet())
	r.Put("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/state", d.handleContentToolsStatePut())
	r.Post("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/submit", d.handleContentToolsStateSubmit())
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

func contentToolsEmptyEnvelope(instanceID uuid.UUID, scope string) ctmodel.ToolStateEnvelope {
	empty := json.RawMessage(`{}`)
	return ctmodel.ToolStateEnvelope{
		InstanceID: instanceID,
		Revision:   0,
		Status:     ctsvc.StatusNotStarted,
		State:      empty,
		Score:      nil,
		UpdatedAt:  nil,
		ResetCount: 0,
		StateJSON:  empty,
		Scope:      scope,
	}
}

func contentToolsStateEnvelope(instanceID uuid.UUID, st *ctrepo.StateRow) ctmodel.ToolStateEnvelope {
	if st == nil {
		return contentToolsEmptyEnvelope(instanceID, ctrepo.ScopeEnrollment)
	}
	state := st.StateJSON
	if len(state) == 0 {
		state = json.RawMessage(`{}`)
	}
	schemaVer := st.StateSchemaVersion
	if schemaVer <= 0 {
		schemaVer = 1
	}
	env := ctmodel.ToolStateEnvelope{
		InstanceID:         instanceID,
		Revision:           st.Revision,
		Status:             st.Status,
		State:              state,
		UpdatedAt:          &st.UpdatedAt,
		ResetCount:         st.ResetCount,
		LastResetAt:        st.LastResetAt,
		StateJSON:          state,
		Scope:              st.Scope,
		StateSchemaVersion: schemaVer,
	}
	if st.ScoreRaw != nil && st.ScoreMax != nil {
		env.Score = &ctmodel.ToolScore{Raw: *st.ScoreRaw, Max: *st.ScoreMax}
	}
	return env
}

// contentToolsStateEnvelopeMigrated applies lazy migration (CT.5) before serving.
func (d Deps) contentToolsStateEnvelopeMigrated(
	ctx context.Context,
	toolID string,
	instanceID uuid.UUID,
	st *ctrepo.StateRow,
) ctmodel.ToolStateEnvelope {
	if st == nil {
		return contentToolsEmptyEnvelope(instanceID, ctrepo.ScopeEnrollment)
	}
	env := contentToolsStateEnvelope(instanceID, st)
	doc, ver, quarantined, err := ctsvc.LazyMigrateState(ctx, d.Pool, toolID, st)
	if err != nil {
		env.Quarantined = true
		return env
	}
	if quarantined {
		env.Quarantined = true
		return env
	}
	env.State = json.RawMessage(doc)
	env.StateJSON = env.State
	env.StateSchemaVersion = ver
	return env
}

func writeContentToolsStateSchemaInvalid(w http.ResponseWriter, err *ctsvc.ConfigValidationError) {
	errs := make([]ctmodel.FieldError, 0, len(err.Errors))
	for _, e := range err.Errors {
		errs = append(errs, ctmodel.FieldError{Path: e.Path, Message: e.Message})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(ctmodel.SchemaInvalidBody{
		Error:  "schema_invalid",
		Errors: errs,
	})
}

func writeContentToolsStateTooLarge(w http.ResponseWriter, maxBytes int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	_ = json.NewEncoder(w).Encode(ctmodel.StateTooLargeBody{
		Error:    "state_too_large",
		MaxBytes: maxBytes,
	})
}

func writeContentToolsRevisionConflict(w http.ResponseWriter, current ctmodel.ToolStateEnvelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(ctmodel.RevisionConflictBody{
		Error:   "revision_conflict",
		Current: current,
	})
}

func contentToolsMaxStateBytes(m *ctsvc.CompiledManifest) int {
	maxBytes := ctsvc.DefaultMaxStateBytes
	if m != nil && m.Storage.MaxStateBytes > 0 && m.Storage.MaxStateBytes < maxBytes {
		maxBytes = m.Storage.MaxStateBytes
	}
	return maxBytes
}

// contentToolsInteractRole maps enrollment roles to manifest interact roles.
func (d Deps) contentToolsInteractRole(ctx context.Context, courseCode string, viewer uuid.UUID) (string, error) {
	staff, err := enrollment.UserIsCourseStaff(ctx, d.Pool, courseCode, viewer)
	if err != nil {
		return "", err
	}
	if staff {
		return "instructor", nil
	}
	roles, err := enrollment.UserRolesInCourse(ctx, d.Pool, courseCode, viewer)
	if err != nil {
		return "", err
	}
	for _, r := range roles {
		switch strings.ToLower(strings.TrimSpace(r)) {
		case "observer", "parent", "guardian":
			return "observer", nil
		}
	}
	return "student", nil
}

// resolveContentToolsStateActor returns enrollment for state read/write.
// Enrollment scope: any active enrollment (students + instructors as self, FR-15).
// Preview scope: editors only.
func (d Deps) resolveContentToolsStateActor(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
	viewer uuid.UUID,
	courseID uuid.UUID,
	scope string,
	m *ctsvc.CompiledManifest,
) (enrollID uuid.UUID, interactRole string, readOnly bool, ok bool) {
	interactRole, err := d.contentToolsInteractRole(r.Context(), courseCode, viewer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.Nil, "", false, false
	}

	if scope == ctrepo.ScopePreview {
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return uuid.Nil, "", false, false
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return uuid.Nil, "", false, false
		}
		id, err := enrollment.GetActiveEnrollmentID(r.Context(), d.Pool, courseID, viewer)
		if err != nil || id == nil {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Active enrollment required.")
			return uuid.Nil, "", false, false
		}
		return *id, interactRole, false, true
	}

	id, err := enrollment.GetActiveEnrollmentID(r.Context(), d.Pool, courseID, viewer)
	if err != nil || id == nil {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Active enrollment required.")
		return uuid.Nil, "", false, false
	}

	mayInteract := false
	if m != nil {
		for _, role := range m.Roles.Interact {
			if role == interactRole {
				mayInteract = true
				break
			}
		}
	}
	if !mayInteract {
		// Observers / denied roles may still read their (empty) state as read-only.
		return *id, interactRole, true, true
	}
	return *id, interactRole, false, true
}

func (d Deps) contentToolsStateRateLimited(w http.ResponseWriter, r *http.Request, userID, instanceID uuid.UUID) bool {
	limiter := d.buildRateLimiter()
	rule := config.RateLimitRule{Limit: ctsvc.StateWriteRateLimitPerMin, Window: time.Minute}
	key := limiter.UserKey(userID.String(), "ct_state_"+instanceID.String())
	dec := limiter.Allow(r.Context(), key, rule, ratelimit.LimitTypeToken)
	if dec.Allowed {
		return false
	}
	ratelimit.RecordExceeded("content_tool_state", ratelimit.LimitTypeToken)
	w.Header().Set("Retry-After", strconv.Itoa(dec.RetryAfter))
	apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many state saves. Try again later.")
	return true
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
		m := ctsvc.MustDefault().Get(inst.ToolID)
		enrollID, _, _, ok := d.resolveContentToolsStateActor(w, r, courseCode, viewer, courseID, scope, m)
		if !ok {
			return
		}

		var st *ctrepo.StateRow
		if scope == ctrepo.ScopePreview {
			st, err = ctrepo.GetStateByScope(r.Context(), d.Pool, instanceID, enrollID, scope)
		} else {
			st, err = ctrepo.GetState(r.Context(), d.Pool, instanceID, enrollID)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load state.")
			return
		}
		env := contentToolsEmptyEnvelope(instanceID, scope)
		if st != nil {
			env = d.contentToolsStateEnvelopeMigrated(r.Context(), inst.ToolID, instanceID, st)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(env)
	}
}

func decodeContentToolsSaveBody(r *http.Request) (ctmodel.SaveStateRequest, error) {
	var body ctmodel.SaveStateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return body, err
	}
	if len(body.State) == 0 && len(body.StateJSON) > 0 {
		body.State = body.StateJSON
	}
	if len(body.StateJSON) == 0 && len(body.State) > 0 {
		body.StateJSON = body.State
	}
	return body, nil
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
		if ctsvc.RuntimeReadonly() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeServiceUnavailable, ctsvc.ErrRuntimeReadonly.Error())
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
		enrollID, _, readOnly, ok := d.resolveContentToolsStateActor(w, r, courseCode, viewer, courseID, scope, m)
		if !ok {
			return
		}
		if readOnly {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if d.contentToolsStateRateLimited(w, r, viewer, instanceID) {
			return
		}

		body, err := decodeContentToolsSaveBody(r)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := ctsvc.ValidateStateJSON(m, body.State); err != nil {
			if err == ctsvc.ErrStateTooLarge {
				ctsvc.IncStateSave(inst.ToolID, "too_large")
				writeContentToolsStateTooLarge(w, contentToolsMaxStateBytes(m))
				return
			}
			if ve, ok := err.(*ctsvc.ConfigValidationError); ok {
				ctsvc.IncStateSave(inst.ToolID, "schema_invalid")
				writeContentToolsStateSchemaInvalid(w, ve)
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		var current *ctrepo.StateRow
		if scope == ctrepo.ScopePreview {
			current, err = ctrepo.GetStateByScope(r.Context(), d.Pool, instanceID, enrollID, scope)
		} else {
			current, err = ctrepo.GetState(r.Context(), d.Pool, instanceID, enrollID)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load state.")
			return
		}
		curStatus := ctsvc.StatusNotStarted
		if current != nil {
			curStatus = current.Status
		}
		nextStatus, err := ctsvc.NextStatusOnSave(curStatus, body.Status)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, ctsvc.ErrInvalidStateStatus.Error())
			return
		}

		schemaVer := ctsvc.DefaultMigrations().CurrentStateSchemaVersion(inst.ToolID)
		var st *ctrepo.StateRow
		if scope == ctrepo.ScopePreview {
			if nextStatus == ctsvc.StatusInProgress && schemaVer <= 1 {
				st, err = ctrepo.UpsertPreviewState(r.Context(), d.Pool, instanceID, enrollID, viewer, body.State, body.Revision)
			} else {
				st, err = ctrepo.UpsertPreviewStateWithStatus(r.Context(), d.Pool, instanceID, enrollID, viewer, body.State, body.Revision, nextStatus, schemaVer)
			}
		} else if nextStatus == ctsvc.StatusInProgress && schemaVer <= 1 {
			st, err = ctrepo.UpsertState(r.Context(), d.Pool, instanceID, enrollID, viewer, body.State, body.Revision)
		} else {
			st, err = ctrepo.UpsertStateWithStatus(r.Context(), d.Pool, instanceID, enrollID, viewer, body.State, body.Revision, nextStatus, schemaVer)
		}
		if err != nil {
			if ctrepo.IsConfigSizeViolation(err) {
				ctsvc.IncStateSave(inst.ToolID, "too_large")
				writeContentToolsStateTooLarge(w, contentToolsMaxStateBytes(m))
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save state.")
			return
		}
		if st == nil {
			ctsvc.IncStateConflict(inst.ToolID)
			ctsvc.IncStateSave(inst.ToolID, "conflict")
			var latest *ctrepo.StateRow
			if scope == ctrepo.ScopePreview {
				latest, _ = ctrepo.GetStateByScope(r.Context(), d.Pool, instanceID, enrollID, scope)
			} else {
				latest, _ = ctrepo.GetState(r.Context(), d.Pool, instanceID, enrollID)
			}
			writeContentToolsRevisionConflict(w, d.contentToolsStateEnvelopeMigrated(r.Context(), inst.ToolID, instanceID, latest))
			return
		}
		ctsvc.IncStateSave(inst.ToolID, "ok")
		actor := viewer
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &instanceID, &enrollID, &actor, inst.ToolID, ctsvc.EventStateSaved, map[string]any{
			"revision":           st.Revision,
			"status":             st.Status,
			"scope":              scope,
			"stateSchemaVersion": st.StateSchemaVersion,
		})
		d.afterContentToolsStateWrite(r.Context(), courseID, courseCode, inst.ToolID, scope, st, "interacted")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(contentToolsStateEnvelope(instanceID, st))
	}
}

func (d Deps) handleContentToolsStateSubmit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, courseID, ok := d.requireContentToolsCourse(w, r)
		if !ok {
			return
		}
		if ctsvc.RuntimeReadonly() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeServiceUnavailable, ctsvc.ErrRuntimeReadonly.Error())
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
		enrollID, _, readOnly, ok := d.resolveContentToolsStateActor(w, r, courseCode, viewer, courseID, scope, m)
		if !ok {
			return
		}
		if readOnly {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		if d.contentToolsStateRateLimited(w, r, viewer, instanceID) {
			return
		}

		body, err := decodeContentToolsSaveBody(r)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := ctsvc.ValidateStateJSON(m, body.State); err != nil {
			if err == ctsvc.ErrStateTooLarge {
				writeContentToolsStateTooLarge(w, contentToolsMaxStateBytes(m))
				return
			}
			if ve, ok := err.(*ctsvc.ConfigValidationError); ok {
				writeContentToolsStateSchemaInvalid(w, ve)
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		var current *ctrepo.StateRow
		if scope == ctrepo.ScopePreview {
			current, err = ctrepo.GetStateByScope(r.Context(), d.Pool, instanceID, enrollID, scope)
		} else {
			current, err = ctrepo.GetState(r.Context(), d.Pool, instanceID, enrollID)
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load state.")
			return
		}
		curStatus := ctsvc.StatusNotStarted
		if current != nil {
			curStatus = current.Status
		}
		nextStatus, err := ctsvc.NextStatusOnSave(curStatus, ctsvc.StatusSubmitted)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, ctsvc.ErrInvalidStateStatus.Error())
			return
		}

		schemaVer := ctsvc.DefaultMigrations().CurrentStateSchemaVersion(inst.ToolID)
		var st *ctrepo.StateRow
		if scope == ctrepo.ScopePreview {
			st, err = ctrepo.UpsertPreviewStateWithStatus(r.Context(), d.Pool, instanceID, enrollID, viewer, body.State, body.Revision, nextStatus, schemaVer)
		} else {
			st, err = ctrepo.UpsertStateWithStatus(r.Context(), d.Pool, instanceID, enrollID, viewer, body.State, body.Revision, nextStatus, schemaVer)
		}
		if err != nil {
			if ctrepo.IsConfigSizeViolation(err) {
				writeContentToolsStateTooLarge(w, contentToolsMaxStateBytes(m))
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to submit state.")
			return
		}
		if st == nil {
			ctsvc.IncStateConflict(inst.ToolID)
			var latest *ctrepo.StateRow
			if scope == ctrepo.ScopePreview {
				latest, _ = ctrepo.GetStateByScope(r.Context(), d.Pool, instanceID, enrollID, scope)
			} else {
				latest, _ = ctrepo.GetState(r.Context(), d.Pool, instanceID, enrollID)
			}
			writeContentToolsRevisionConflict(w, d.contentToolsStateEnvelopeMigrated(r.Context(), inst.ToolID, instanceID, latest))
			return
		}
		ctsvc.IncStateSave(inst.ToolID, "submitted")
		actor := viewer
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &instanceID, &enrollID, &actor, inst.ToolID, ctsvc.EventStateSubmitted, map[string]any{
			"revision": st.Revision,
			"status":   st.Status,
		})
		d.afterContentToolsStateWrite(r.Context(), courseID, courseCode, inst.ToolID, scope, st, "completed")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(contentToolsStateEnvelope(instanceID, st))
	}
}
