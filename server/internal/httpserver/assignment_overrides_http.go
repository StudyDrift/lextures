package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/assignmentoverrides"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// Plan 2.15: instructor-facing "assign to" editor API. One item_id covers both assignments
// and quizzes since both live in course.course_structure_items / course.assignment_overrides.

func targetJSON(t *assignmentoverrides.Target) map[string]any {
	out := map[string]any{
		"id":         t.ID.String(),
		"targetType": t.TargetType,
		"createdAt":  t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if t.TargetID != nil {
		out["targetId"] = t.TargetID.String()
	}
	if t.DueAt != nil {
		out["dueAt"] = t.DueAt.UTC().Format(time.RFC3339Nano)
	}
	if t.AvailableFrom != nil {
		out["availableFrom"] = t.AvailableFrom.UTC().Format(time.RFC3339Nano)
	}
	if t.AvailableUntil != nil {
		out["availableUntil"] = t.AvailableUntil.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func (d Deps) requireOverridesEditableItem(w http.ResponseWriter, r *http.Request) (courseID uuid.UUID, itemID uuid.UUID, viewer uuid.UUID, itemKind string, ok bool) {
	courseCode, v, ok := d.requireCourseAccess(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, "", false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, "", false
	}
	iid, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, "", false
	}
	row, err := coursestructure.GetItemRow(r.Context(), d.Pool, *cid, iid)
	if err != nil || row == nil || (row.Kind != "assignment" && row.Kind != "quiz") {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Assignment or quiz not found in course.")
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, "", false
	}
	can, err := courseroles.UserHasPermission(r.Context(), d.Pool, v, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, "", false
	}
	if !can {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage assign-to targets.")
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, "", false
	}
	return *cid, iid, v, row.Kind, true
}

func (d Deps) handleAssignmentOverridesCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseID, itemID, viewer, itemKind, ok := d.requireOverridesEditableItem(w, r)
		if !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			targets, err := assignmentoverrides.ListForItem(r.Context(), d.Pool, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load assign-to targets.")
				return
			}
			orphaned, err := assignmentoverrides.IsOrphaned(r.Context(), d.Pool, courseID, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to evaluate assign-to targets.")
				return
			}
			arr := make([]map[string]any, 0, len(targets))
			for i := range targets {
				arr = append(arr, targetJSON(&targets[i]))
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"targets": arr, "orphaned": orphaned})
			return

		case http.MethodPut:
			var body struct {
				Targets []struct {
					TargetType     string     `json:"targetType"`
					TargetID       *string    `json:"targetId"`
					DueAt          *time.Time `json:"dueAt"`
					AvailableFrom  *time.Time `json:"availableFrom"`
					AvailableUntil *time.Time `json:"availableUntil"`
				} `json:"targets"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			writes := make([]assignmentoverrides.TargetWrite, 0, len(body.Targets))
			invalidTarget := false
			for _, t := range body.Targets {
				tw := assignmentoverrides.TargetWrite{
					TargetType: t.TargetType, DueAt: t.DueAt, AvailableFrom: t.AvailableFrom, AvailableUntil: t.AvailableUntil,
				}
				if t.TargetID != nil && *t.TargetID != "" {
					tid, err := uuid.Parse(*t.TargetID)
					if err != nil {
						invalidTarget = true
						break
					}
					tw.TargetID = &tid
				}
				writes = append(writes, tw)
			}
			if invalidTarget {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid targetId.")
				return
			}
			if err := assignmentoverrides.ReplaceForItem(r.Context(), d.Pool, itemID, itemKind, writes, viewer); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not save assign-to targets: "+err.Error())
				return
			}
			targets, err := assignmentoverrides.ListForItem(r.Context(), d.Pool, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reload assign-to targets.")
				return
			}
			orphaned, err := assignmentoverrides.IsOrphaned(r.Context(), d.Pool, courseID, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to evaluate assign-to targets.")
				return
			}
			arr := make([]map[string]any, 0, len(targets))
			for i := range targets {
				arr = append(arr, targetJSON(&targets[i]))
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"targets": arr, "orphaned": orphaned})
			return

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

func (d Deps) handleAssignmentOverridesBulkExtend() http.HandlerFunc {
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
		_, itemID, viewer, itemKind, ok := d.requireOverridesEditableItem(w, r)
		if !ok {
			return
		}
		var body struct {
			EnrollmentIDs []string  `json:"enrollmentIds"`
			DueAt         time.Time `json:"dueAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(body.EnrollmentIDs) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "enrollmentIds is required.")
			return
		}
		if body.DueAt.IsZero() {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "dueAt is required.")
			return
		}
		ids := make([]uuid.UUID, 0, len(body.EnrollmentIDs))
		for _, s := range body.EnrollmentIDs {
			eid, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
				return
			}
			ids = append(ids, eid)
		}
		if err := assignmentoverrides.BulkExtendDueDate(r.Context(), d.Pool, itemID, itemKind, ids, body.DueAt, viewer); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to extend due dates.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}
