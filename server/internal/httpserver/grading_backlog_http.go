package httpserver

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
)

// handleCourseGradingBacklog is GET /api/v1/courses/{course_code}/grading-backlog.
func (d Deps) handleCourseGradingBacklog() http.HandlerFunc {
	type item struct {
		ItemID          string `json:"itemId"`
		ItemType        string `json:"itemType"`
		AssignmentID    string `json:"assignmentId"`
		AssignmentTitle string `json:"assignmentTitle"`
		UngradedCount   int64  `json:"ungradedCount"`
	}
	type resp struct {
		Items []item `json:"items"`
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
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view the grading backlog.")
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
		assignRows, err := moduleassignmentsubmissions.ListUngradedCountsForCourse(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grading backlog.")
			return
		}
		quizRows, err := quizattempts.ListUngradedCountsForCourse(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load grading backlog.")
			return
		}
		items := make([]item, 0, len(assignRows)+len(quizRows))
		for _, row := range assignRows {
			id := row.ModuleItemID.String()
			items = append(items, item{
				ItemID:          id,
				ItemType:        "assignment",
				AssignmentID:    id,
				AssignmentTitle: row.Title,
				UngradedCount:   row.UngradedCount,
			})
		}
		for _, row := range quizRows {
			id := row.StructureItemID.String()
			items = append(items, item{
				ItemID:          id,
				ItemType:        "quiz",
				AssignmentID:    id,
				AssignmentTitle: row.Title,
				UngradedCount:   row.UngradedCount,
			})
		}
		sort.Slice(items, func(i, j int) bool {
			if items[i].UngradedCount != items[j].UngradedCount {
				return items[i].UngradedCount > items[j].UngradedCount
			}
			if items[i].AssignmentTitle != items[j].AssignmentTitle {
				return items[i].AssignmentTitle < items[j].AssignmentTitle
			}
			return items[i].ItemType < items[j].ItemType
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Items: items})
	}
}