package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	modelenrollment "github.com/lextures/lextures/server/internal/models/enrollment"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

func (d Deps) requireEnrollmentStateMachine(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFEnrollmentStateMachine && !d.Config.FFEnrollmentStateMachine {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Enrollment state machine is not enabled.")
		return false
	}
	return true
}

func (d Deps) handleEnrollmentStatePatch() http.HandlerFunc {
	type reqBody struct {
		State            string  `json:"state"`
		Reason           *string `json:"reason"`
		OverrideDeadline *bool   `json:"overrideDeadline"`
	}
	type respBody struct {
		ID             string  `json:"id"`
		State          string  `json:"state"`
		StateChangedAt *string `json:"stateChangedAt,omitempty"`
		StateReason    *string `json:"stateReason,omitempty"`
		LISStatusCode  string  `json:"lisStatusCode"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireEnrollmentStateMachine(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		enrollIDStr := strings.TrimSpace(chi.URLParam(r, "enrollment_id"))
		enrollID, err := uuid.Parse(enrollIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		canUpdate, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":enrollments:update")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canUpdate {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to change enrollment state.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		newState, err := modelenrollment.ParseState(body.State)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid state.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		existing, err := enrollment.GetStateByID(r.Context(), d.Pool, *cid, enrollID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
			return
		}
		if existing.Role != "student" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "State changes apply to student enrollments only.")
			return
		}

		override := body.OverrideDeadline != nil && *body.OverrideDeadline
		if override {
			ga, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, permGlobalRBACManage)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
				return
			}
			if !ga {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only global admins may override enrollment deadlines.")
				return
			}
		}

		deadlines, err := enrollment.TermDeadlinesForCourse(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load term deadlines.")
			return
		}
		dc := modelenrollment.DeadlineContext{
			AddDropDeadline:    deadlines.AddDropDeadline,
			WithdrawalDeadline: deadlines.WithdrawalDeadline,
			OverrideDeadlines:  override,
		}
		actor := viewer
		updated, err := enrollment.TransitionState(
			r.Context(), d.Pool, enrollID, *cid, &actor, newState, body.Reason, "manual", dc,
		)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "deadline") || strings.Contains(msg, "already") {
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, msg)
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update enrollment state.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
			return
		}
		out := respBody{
			ID:            updated.ID.String(),
			State:         string(updated.State),
			StateReason:   updated.StateReason,
			LISStatusCode: updated.State.LISStatusCode(),
		}
		if updated.StateChangedAt != nil {
			s := updated.StateChangedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			out.StateChangedAt = &s
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handleEnrollmentStateHistory() http.HandlerFunc {
	type histRow struct {
		ID            string  `json:"id"`
		ActorID       *string `json:"actorId,omitempty"`
		PreviousState string  `json:"previousState"`
		NewState      string  `json:"newState"`
		Reason        *string `json:"reason,omitempty"`
		Source        string  `json:"source"`
		CreatedAt     string  `json:"createdAt"`
	}
	type respBody struct {
		History []histRow `json:"history"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireEnrollmentStateMachine(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		enrollIDStr := strings.TrimSpace(chi.URLParam(r, "enrollment_id"))
		enrollID, err := uuid.Parse(enrollIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		canRead, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":enrollments:read")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canRead {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view enrollment history.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		existing, err := enrollment.GetStateByID(r.Context(), d.Pool, *cid, enrollID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
			return
		}
		history, err := enrollment.ListStateHistory(r.Context(), d.Pool, enrollID, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load history.")
			return
		}
		out := make([]histRow, 0, len(history))
		for _, h := range history {
			row := histRow{
				ID:            h.ID.String(),
				PreviousState: string(h.PreviousState),
				NewState:      string(h.NewState),
				Reason:        h.Reason,
				Source:        h.Source,
				CreatedAt:     h.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			}
			if h.ActorID != nil {
				s := h.ActorID.String()
				row.ActorID = &s
			}
			out = append(out, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(respBody{History: out})
	}
}
