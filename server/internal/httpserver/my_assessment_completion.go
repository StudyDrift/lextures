package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/studentprogress"
)

// handleMyAssessmentCompletion is GET /api/v1/courses/{course_code}/my-assessment-completion.
// Students receive the structure item ids of assignments they have submitted and
// quizzes they have submitted an attempt for. Other viewers receive an empty list.
func (d Deps) handleMyAssessmentCompletion() http.HandlerFunc {
	type resp struct {
		CompletedItemIDs []string `json:"completedItemIds"`
	}
	write := func(w http.ResponseWriter, ids []string) {
		if ids == nil {
			ids = []string{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{CompletedItemIDs: ids})
	}
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		isStudent, err := enrollment.UserHasStudentEquivalentEnrollment(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify enrollment.")
			return
		}
		if !isStudent {
			write(w, []string{})
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
		ids, err := studentprogress.ListCompletedAssessmentItemIDs(r.Context(), d.Pool, *cid, viewer)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load assessment completion.", err)
			return
		}
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			out = append(out, id.String())
		}
		write(w, out)
	}
}
