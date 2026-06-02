package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/gradeauditevents"
)

// handleGetAssignmentGradeHistory is GET /api/v1/courses/{course_code}/assignments/{item_id}/grades/{student_id}/history.
func (d Deps) handleGetAssignmentGradeHistory() http.HandlerFunc {
	type eventOut struct {
		ID             string   `json:"id"`
		Action         string   `json:"action"`
		PreviousScore  *float64 `json:"previousScore"`
		NewScore       *float64 `json:"newScore"`
		PreviousStatus *string  `json:"previousStatus"`
		NewStatus      *string  `json:"newStatus"`
		Reason         *string  `json:"reason"`
		ChangedAt      string   `json:"changedAt"`
		ChangedBy      *string  `json:"changedBy"`
	}
	type resp struct {
		Events []eventOut `json:"events"`
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

		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}

		has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view grade history.")
			return
		}

		assignmentID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment ID.")
			return
		}
		studentID, err := uuid.Parse(chi.URLParam(r, "student_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student ID.")
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

		rows, err := gradeauditevents.ListForCell(r.Context(), d.Pool, *cid, assignmentID, studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grade history.")
			return
		}

		events := make([]eventOut, 0, len(rows))
		for _, row := range rows {
			e := eventOut{
				ID:             row.ID.String(),
				Action:         row.Action,
				PreviousScore:  row.PreviousScore,
				NewScore:       row.NewScore,
				PreviousStatus: row.PreviousStatus,
				NewStatus:      row.NewStatus,
				Reason:         row.Reason,
				ChangedAt:      row.ChangedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
			}
			if row.ChangedBy != nil {
				s := row.ChangedBy.String()
				e.ChangedBy = &s
			}
			events = append(events, e)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Events: events})
	}
}
