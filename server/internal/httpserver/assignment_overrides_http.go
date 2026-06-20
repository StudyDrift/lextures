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

type assignToTargetJSON struct {
	TargetType     string     `json:"targetType"`
	TargetID       *string    `json:"targetId,omitempty"`
	DueAt          *time.Time `json:"dueAt,omitempty"`
	AvailableFrom  *time.Time `json:"availableFrom,omitempty"`
	AvailableUntil *time.Time `json:"availableUntil,omitempty"`
	ExtraAttempts  *int32     `json:"extraAttempts,omitempty"`
	TimeMultiplier *float64   `json:"timeMultiplier,omitempty"`
}

func assignToTargetsToJSON(rows []assignmentoverrides.OverrideRow) []assignToTargetJSON {
	out := make([]assignToTargetJSON, 0, len(rows))
	for _, r := range rows {
		t := assignToTargetJSON{
			TargetType:     r.TargetType,
			DueAt:          r.DueAt,
			AvailableFrom:  r.AvailableFrom,
			AvailableUntil: r.AvailableUntil,
			ExtraAttempts:  r.ExtraAttempts,
			TimeMultiplier: r.TimeMultiplier,
		}
		if r.TargetID != nil {
			s := r.TargetID.String()
			t.TargetID = &s
		}
		out = append(out, t)
	}
	return out
}

// resolveItemForOverrides loads the course id and validates the item is an assignment/quiz in the course.
func (d Deps) resolveItemForOverrides(w http.ResponseWriter, r *http.Request, courseCode string, itemID uuid.UUID) (uuid.UUID, bool) {
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return uuid.UUID{}, false
	}
	row, err := coursestructure.GetItemRow(r.Context(), d.Pool, *cid, itemID)
	if err != nil || row == nil || (row.Kind != "assignment" && row.Kind != "quiz") {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Item not found in course.")
		return uuid.UUID{}, false
	}
	return *cid, true
}

// handleItemAssignToOverrides is GET/PUT /api/v1/courses/{course_code}/items/{item_id}/overrides.
// Instructor-only: lists or replaces the assign-to targets (everyone/section/group/student) for one
// assignment/quiz item (plan 2.15).
func (d Deps) handleItemAssignToOverrides() http.HandlerFunc {
	type listResp struct {
		Targets  []assignToTargetJSON `json:"targets"`
		Orphaned bool                 `json:"orphaned"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !d.effectiveConfig().FFAssignToOverrides {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Assign-to targeting is not enabled for this platform.")
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		can, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !can {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage assign-to targeting.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		if _, ok := d.resolveItemForOverrides(w, r, courseCode, itemID); !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			rows, err := assignmentoverrides.ListForItem(r.Context(), d.Pool, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load assign-to targets.")
				return
			}
			orphaned, err := assignmentoverrides.HasOrphanedTargeting(r.Context(), d.Pool, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check assign-to targeting.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(listResp{Targets: assignToTargetsToJSON(rows), Orphaned: orphaned})
			return

		case http.MethodPut:
			var body struct {
				Targets []assignToTargetJSON `json:"targets"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			writes := make([]assignmentoverrides.OverrideWrite, 0, len(body.Targets))
			for _, t := range body.Targets {
				var targetID *uuid.UUID
				if t.TargetID != nil && *t.TargetID != "" {
					id, perr := uuid.Parse(*t.TargetID)
					if perr != nil {
						apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid targetId.")
						return
					}
					targetID = &id
				}
				writes = append(writes, assignmentoverrides.OverrideWrite{
					TargetType:     t.TargetType,
					TargetID:       targetID,
					DueAt:          t.DueAt,
					AvailableFrom:  t.AvailableFrom,
					AvailableUntil: t.AvailableUntil,
					ExtraAttempts:  t.ExtraAttempts,
					TimeMultiplier: t.TimeMultiplier,
				})
			}
			if err := assignmentoverrides.ReplaceForItem(r.Context(), d.Pool, itemID, writes, viewer); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not save assign-to targets: "+err.Error())
				return
			}
			rows, err := assignmentoverrides.ListForItem(r.Context(), d.Pool, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load assign-to targets.")
				return
			}
			orphaned, err := assignmentoverrides.HasOrphanedTargeting(r.Context(), d.Pool, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check assign-to targeting.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(listResp{Targets: assignToTargetsToJSON(rows), Orphaned: orphaned})
			return

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPut+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleItemAssignToBulkExtend is POST /api/v1/courses/{course_code}/items/{item_id}/overrides/bulk-extend.
// Instructor-only: sets a student-level due date override for each selected enrollment (plan 2.15 FR-8).
func (d Deps) handleItemAssignToBulkExtend() http.HandlerFunc {
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
		if !d.effectiveConfig().FFAssignToOverrides {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Assign-to targeting is not enabled for this platform.")
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		can, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !can {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage assign-to targeting.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		if _, ok := d.resolveItemForOverrides(w, r, courseCode, itemID); !ok {
			return
		}
		var body struct {
			EnrollmentIDs []string   `json:"enrollmentIds"`
			DueAt         *time.Time `json:"dueAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.DueAt == nil || len(body.EnrollmentIDs) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "dueAt and enrollmentIds are required.")
			return
		}
		existing, err := assignmentoverrides.ListForItem(r.Context(), d.Pool, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load assign-to targets.")
			return
		}
		extended := make(map[uuid.UUID]bool, len(body.EnrollmentIDs))
		writes := make([]assignmentoverrides.OverrideWrite, 0, len(existing)+len(body.EnrollmentIDs))
		for _, idStr := range body.EnrollmentIDs {
			id, perr := uuid.Parse(idStr)
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollmentIds.")
				return
			}
			extended[id] = true
		}
		for _, r := range existing {
			if r.TargetType == assignmentoverrides.TargetStudent && r.TargetID != nil && extended[*r.TargetID] {
				continue // replaced below with the extended due date
			}
			writes = append(writes, assignmentoverrides.OverrideWrite{
				TargetType: r.TargetType, TargetID: r.TargetID,
				DueAt: r.DueAt, AvailableFrom: r.AvailableFrom, AvailableUntil: r.AvailableUntil,
				ExtraAttempts: r.ExtraAttempts, TimeMultiplier: r.TimeMultiplier,
			})
		}
		for id := range extended {
			eid := id
			due := *body.DueAt
			writes = append(writes, assignmentoverrides.OverrideWrite{
				TargetType: assignmentoverrides.TargetStudent, TargetID: &eid, DueAt: &due,
			})
		}
		if err := assignmentoverrides.ReplaceForItem(r.Context(), d.Pool, itemID, writes, viewer); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not extend due date: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}
}
