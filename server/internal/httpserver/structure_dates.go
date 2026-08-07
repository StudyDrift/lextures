package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	coursestructurerepo "github.com/lextures/lextures/server/internal/repos/coursestructure"
)

const (
	maxBulkDueAtUpdates = 500
)

// handlePostBulkStructureDueAt is POST /api/v1/courses/{course_code}/structure/dates/bulk
func (d Deps) handlePostBulkStructureDueAt() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit course structure.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		var body struct {
			Updates []struct {
				ItemID string     `json:"itemId"`
				DueAt  *time.Time `json:"dueAt"`
			} `json:"updates"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(body.Updates) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "updates is required.")
			return
		}
		if len(body.Updates) > maxBulkDueAtUpdates {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				fmt.Sprintf("Too many updates (max %d).", maxBulkDueAtUpdates))
			return
		}

		updates := make([]coursestructurerepo.DueAtUpdate, 0, len(body.Updates))
		for _, u := range body.Updates {
			id, parseErr := uuid.Parse(strings.TrimSpace(u.ItemID))
			if parseErr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid itemId in updates.")
				return
			}
			if u.DueAt == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "dueAt is required for each update.")
				return
			}
			t := u.DueAt.UTC()
			updates = append(updates, coursestructurerepo.DueAtUpdate{ItemID: id, DueAt: &t})
		}

		updated, failed, err := coursestructurerepo.BulkPatchChildDueAt(r.Context(), d.Pool, *cid, updates)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update due dates.")
			return
		}
		d.invalidateCourseStructureCache(r.Context(), *cid)
		if d.calendarFeedsEnabled() {
			d.invalidateCourseCalendarCache(r.Context(), *cid)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]int{
			"updated": updated,
			"failed":  failed,
		})
	}
}
