package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// handlePatchCourseGradebookCellExcused is PATCH /api/v1/courses/{course_code}/gradebook/cells/{item_id}/excused.
// Same permission as other gradebook cell edits: course:{code}:item:create.
func (d Deps) handlePatchCourseGradebookCellExcused() http.HandlerFunc {
	type body struct {
		StudentID string  `json:"studentId"`
		Excused   bool    `json:"excused"`
		Reason    *string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}

		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit grades.")
			return
		}

		itemID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "item_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}

		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var b body
		if err := json.Unmarshal(payload, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(b.StudentID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
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

		if err := coursegrades.SetExcusedForCell(r.Context(), d.Pool, *cid, viewer, studentID, itemID, b.Excused, b.Reason); err != nil {
			if errors.Is(err, coursegrades.ErrGradebookItemNotInCourse) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Assignment is not in this course.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update excused status.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
