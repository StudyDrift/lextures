package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/coursestructure"
	coursestructurerepo "github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/courseroles"
)

// handleReorderCourseStructure is POST /api/v1/courses/{course_code}/structure/reorder.
func (d Deps) handleReorderCourseStructure() http.HandlerFunc {
	type resp struct {
		Items []coursestructurerepo.ItemResponse `json:"items"`
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
		var body coursestructure.ReorderCourseStructureRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		err = coursestructurerepo.ApplyModuleAndChildOrder(
			r.Context(), d.Pool, *cid, body.ModuleOrder, body.ChildOrderByModule,
		)
		if errors.Is(err, coursestructurerepo.ErrInvalidReorder) {
			apierr.WriteJSON(
				w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"Invalid reorder: module or child ids must match the current structure.",
			)
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reorder course structure.")
			return
		}
		items, err := coursestructurerepo.ListForCourseWithEnrichment(r.Context(), d.Pool, *cid, true)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course structure.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{Items: items})
	}
}
