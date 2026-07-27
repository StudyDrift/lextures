package httpserver

import (
	"encoding/json"
	"errors"
	"io"
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
	"github.com/lextures/lextures/server/internal/ratelimit"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) registerContentToolsActionRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/actions/{action}", d.handleContentToolsActionRun())
}

func (d Deps) contentToolsActionRateLimited(
	w http.ResponseWriter,
	r *http.Request,
	userID, instanceID uuid.UUID,
	action string,
	limit int,
) bool {
	if limit <= 0 {
		limit = ctsvc.DefaultActionRateLimitPerMin
	}
	limiter := d.buildRateLimiter()
	rule := config.RateLimitRule{Limit: limit, Window: time.Minute}
	key := limiter.UserKey(userID.String(), "ct_action_"+instanceID.String()+"_"+action)
	dec := limiter.Allow(r.Context(), key, rule, ratelimit.LimitTypeToken)
	if dec.Allowed {
		return false
	}
	ratelimit.RecordExceeded("content_tool_action", ratelimit.LimitTypeToken)
	w.Header().Set("Retry-After", strconv.Itoa(dec.RetryAfter))
	apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many action requests. Try again later.")
	return true
}

func (d Deps) handleContentToolsActionRun() http.HandlerFunc {
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
		actionName := strings.TrimSpace(chi.URLParam(r, "action"))
		if actionName == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Action is required.")
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
		decl := ctsvc.FindAction(m, actionName)
		if decl == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Action not found.")
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
		if d.contentToolsActionRateLimited(w, r, viewer, instanceID, actionName, ctsvc.EffectiveActionRateLimit(m, decl)) {
			return
		}

		var body ctmodel.RunActionRequest
		decErr := json.NewDecoder(r.Body).Decode(&body)
		if decErr != nil && !errors.Is(decErr, io.EOF) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(body.Input) == 0 {
			body.Input = json.RawMessage(`{}`)
		}

		idemKey := strings.TrimSpace(body.IdempotencyKey)
		if idemKey != "" {
			prior, err := ctrepo.GetActionIdempotency(r.Context(), d.Pool, idemKey)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check idempotency.")
				return
			}
			if prior != nil && prior.InstanceID == instanceID && prior.EnrollmentID == enrollID && prior.Action == actionName {
				var cached ctmodel.RunActionResponse
				if err := json.Unmarshal(prior.ResultJSON, &cached); err == nil {
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					_ = json.NewEncoder(w).Encode(cached)
					return
				}
			}
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
		stateJSON := json.RawMessage(`{}`)
		status := ctsvc.StatusNotStarted
		revision := int64(0)
		if current != nil {
			stateJSON = current.StateJSON
			status = current.Status
			revision = current.Revision
		}

		start := time.Now()
		result, err := ctsvc.DispatchAction(m, actionName, ctsvc.ActionContext{
			Ctx:          r.Context(),
			CourseID:     courseID,
			CourseCode:   courseCode,
			InstanceID:   instanceID,
			EnrollmentID: enrollID,
			PrincipalID:  viewer,
			ToolID:       inst.ToolID,
			ConfigJSON:   inst.ConfigJSON,
			StateJSON:    stateJSON,
			Status:       status,
			Revision:     revision,
			Input:        body.Input,
		})
		ctsvc.ObserveActionLatency(inst.ToolID, actionName, time.Since(start).Seconds())
		if err != nil {
			ctsvc.DefaultBreaker().RecordFailure(inst.ToolID, err.Error(), time.Now().UTC())
			if errors.Is(err, ctsvc.ErrActionUnknown) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Action not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if result == nil {
			result = &ctsvc.ActionResult{Result: map[string]any{}}
		}
		if result.Result == nil {
			result.Result = map[string]any{}
		}

		nextState := stateJSON
		if len(result.StatePatch) > 0 {
			merged, mergeErr := ctsvc.MergeStateJSON(stateJSON, result.StatePatch)
			if mergeErr != nil {
				nextState = result.StatePatch
			} else {
				nextState = merged
			}
		}
		if err := ctsvc.ValidateStateJSON(m, nextState); err != nil {
			if err == ctsvc.ErrStateTooLarge {
				writeContentToolsStateTooLarge(w, contentToolsMaxStateBytes(m))
				return
			}
			if ve, ok := err.(*ctsvc.ConfigValidationError); ok {
				writeContentToolsStateSchemaInvalid(w, ve)
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Action produced invalid state.")
			return
		}

		nextStatus := status
		if result.Status != "" {
			if !ctsvc.CanTransitionStateStatus(status, result.Status) {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Action produced invalid status.")
				return
			}
			nextStatus = result.Status
		} else if nextStatus == ctsvc.StatusNotStarted {
			nextStatus = ctsvc.StatusInProgress
		}

		// Auto scoring: only server actions may set scores (never client PUT).
		var scoreRaw, scoreMax *float64
		if m.Scoring.Mode == "auto" {
			scoreRaw = result.ScoreRaw
			scoreMax = result.ScoreMax
			if scoreMax == nil && m.Scoring.MaxScore != nil {
				scoreMax = m.Scoring.MaxScore
			}
		}

		st, err := ctrepo.ApplyActionState(
			r.Context(), d.Pool, instanceID, enrollID, viewer,
			nextState, revision, nextStatus, scoreRaw, scoreMax, scope,
		)
		if err != nil {
			if ctrepo.IsConfigSizeViolation(err) {
				writeContentToolsStateTooLarge(w, contentToolsMaxStateBytes(m))
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to apply action.")
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
			writeContentToolsRevisionConflict(w, contentToolsStateEnvelope(instanceID, latest))
			return
		}

		resp := ctmodel.RunActionResponse{
			Result: result.Result,
			State:  contentToolsStateEnvelope(instanceID, st),
		}
		if idemKey != "" {
			raw, _ := json.Marshal(resp)
			_ = ctrepo.PutActionIdempotency(r.Context(), d.Pool, idemKey, instanceID, enrollID, actionName, raw)
		}
		actor := viewer
		_ = ctrepo.InsertEvent(r.Context(), d.Pool, courseID, &instanceID, &enrollID, &actor, inst.ToolID, ctsvc.EventActionRan, map[string]any{
			"action":   actionName,
			"revision": st.Revision,
			"status":   st.Status,
		})
		ctsvc.DefaultBreaker().RecordSuccess(inst.ToolID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
