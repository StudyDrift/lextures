package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	repo "github.com/lextures/lextures/server/internal/repos/incompletegrades"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	svc "github.com/lextures/lextures/server/internal/service/incompletegrades"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

func (d Deps) requireIncompleteGradeWorkflow(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFIncompleteGradeWorkflow && !d.effectiveConfig().FFIncompleteGradeWorkflow {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Incomplete grade workflow is not enabled.")
		return false
	}
	if !cfg.FFEnrollmentStateMachine && !d.effectiveConfig().FFEnrollmentStateMachine {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Enrollment state machine is required for incomplete grades.")
		return false
	}
	return true
}

func (d Deps) incompleteGradeService() *svc.Service {
	return &svc.Service{
		Pool: d.Pool,
		Notify: &notifications.Service{Pool: d.Pool, Config: d.effectiveConfig()},
	}
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(s))
}

func incompleteRecordToJSON(rec *repo.Record) map[string]any {
	if rec == nil {
		return nil
	}
	itemIDs := make([]string, 0, len(rec.OutstandingItemIDs))
	for _, id := range rec.OutstandingItemIDs {
		itemIDs = append(itemIDs, id.String())
	}
	out := map[string]any{
		"id":                 rec.ID.String(),
		"enrollmentId":       rec.EnrollmentID.String(),
		"grantedBy":          rec.GrantedBy.String(),
		"extensionDeadline":  rec.ExtensionDeadline.Format("2006-01-02"),
		"outstandingItemIds": itemIDs,
		"status":             string(rec.Status),
		"createdAt":          rec.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if rec.Notes != nil {
		out["notes"] = *rec.Notes
	}
	if rec.ResolvedGrade != nil {
		out["resolvedGrade"] = *rec.ResolvedGrade
	}
	if rec.ResolvedAt != nil {
		out["resolvedAt"] = rec.ResolvedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if rec.ResolvedBy != nil {
		out["resolvedBy"] = rec.ResolvedBy.String()
	}
	return out
}

func (d Deps) handleIncompleteGradeGet() http.HandlerFunc {
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
		if !d.requireIncompleteGradeWorkflow(w) {
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
		enrollID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "enrollment_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		canRead, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil || !canRead {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view incomplete grades.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		existing, err := enrollment.GetStateByID(r.Context(), d.Pool, *cid, enrollID)
		if err != nil || existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
			return
		}
		rec, err := repo.GetByEnrollmentID(r.Context(), d.Pool, enrollID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load incomplete record.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if rec == nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"record": nil})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"record": incompleteRecordToJSON(rec)})
	}
}

func (d Deps) handleIncompleteGradePost() http.HandlerFunc {
	type reqBody struct {
		ExtensionDeadline  string   `json:"extensionDeadline"`
		OutstandingItemIDs []string `json:"outstandingItemIds"`
		Notes              *string  `json:"notes"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireIncompleteGradeWorkflow(w) {
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
		enrollID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "enrollment_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		canUpdate, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !canUpdate {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to grant incomplete grades.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		deadline, err := parseDate(body.ExtensionDeadline)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid extension deadline (use YYYY-MM-DD).")
			return
		}
		var itemIDs []uuid.UUID
		for _, s := range body.OutstandingItemIDs {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid outstanding item id.")
				return
			}
			itemIDs = append(itemIDs, id)
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rec, err := d.incompleteGradeService().Grant(r.Context(), svc.GrantParams{
			CourseID:           *cid,
			EnrollmentID:       enrollID,
			ActorID:            viewer,
			ExtensionDeadline:  deadline,
			OutstandingItemIDs: itemIDs,
			Notes:              body.Notes,
		})
		if err != nil {
			msg := err.Error()
			switch err {
			case svc.ErrAlreadyExists, svc.ErrInvalidDeadline, svc.ErrOutstandingEmpty:
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, msg)
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to grant incomplete grade.")
			}
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"record": incompleteRecordToJSON(rec)})
	}
}

func (d Deps) handleIncompleteGradePatch() http.HandlerFunc {
	type reqBody struct {
		ResolvedGrade     *string `json:"resolvedGrade"`
		ExtensionDeadline *string `json:"extensionDeadline"`
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
		if !d.requireIncompleteGradeWorkflow(w) {
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
		enrollID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "enrollment_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		canUpdate, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil || !canUpdate {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to update incomplete grades.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		s := d.incompleteGradeService()
		var rec *repo.Record
		switch {
		case body.ResolvedGrade != nil:
			rec, err = s.Resolve(r.Context(), *cid, enrollID, viewer, *body.ResolvedGrade)
		case body.ExtensionDeadline != nil:
			deadline, perr := parseDate(*body.ExtensionDeadline)
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid extension deadline (use YYYY-MM-DD).")
				return
			}
			rec, err = s.ExtendDeadline(r.Context(), *cid, enrollID, viewer, deadline)
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide resolvedGrade or extensionDeadline.")
			return
		}
		if err != nil {
			msg := err.Error()
			switch err {
			case svc.ErrNotOpen, svc.ErrInvalidDeadline, svc.ErrInvalidGrade:
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, msg)
			case svc.ErrNotFound:
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, msg)
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update incomplete grade.")
			}
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"record": incompleteRecordToJSON(rec)})
	}
}

func (d Deps) handleAdminIncompletes() http.HandlerFunc {
	type rowOut struct {
		ID                  string   `json:"id"`
		EnrollmentID        string   `json:"enrollmentId"`
		StudentUserID       string   `json:"studentUserId"`
		StudentName         string   `json:"studentName"`
		CourseCode          string   `json:"courseCode"`
		CourseTitle         string   `json:"courseTitle"`
		ExtensionDeadline   string   `json:"extensionDeadline"`
		OutstandingItemIDs  []string `json:"outstandingItemIds"`
		OutstandingTitles   []string `json:"outstandingTitles"`
		Status              string   `json:"status"`
		Notes               *string  `json:"notes,omitempty"`
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
		if !d.requireIncompleteGradeWorkflow(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Registrar or admin access required.")
			return
		}
		status := repo.StatusOpen
		if q := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status"))); q != "" {
			switch q {
			case "open", "resolved", "lapsed":
				status = repo.Status(q)
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status filter.")
				return
			}
		}
		var termID *uuid.UUID
		if q := strings.TrimSpace(r.URL.Query().Get("term_id")); q != "" {
			tid, err := uuid.Parse(q)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid term_id.")
				return
			}
			termID = &tid
		}
		rows, err := repo.ListReport(r.Context(), d.Pool, termID, status)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load incomplete report.")
			return
		}
		out := make([]rowOut, 0, len(rows))
		for _, rr := range rows {
			itemIDs := make([]string, 0, len(rr.OutstandingItemIDs))
			for _, id := range rr.OutstandingItemIDs {
				itemIDs = append(itemIDs, id.String())
			}
			out = append(out, rowOut{
				ID:                 rr.ID.String(),
				EnrollmentID:       rr.EnrollmentID.String(),
				StudentUserID:      rr.StudentUserID.String(),
				StudentName:        rr.StudentName,
				CourseCode:         rr.CourseCode,
				CourseTitle:        rr.CourseTitle,
				ExtensionDeadline:  rr.ExtensionDeadline.Format("2006-01-02"),
				OutstandingItemIDs: itemIDs,
				OutstandingTitles:  rr.OutstandingTitles,
				Status:             string(rr.Status),
				Notes:              rr.Notes,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"incompletes": out})
	}
}
