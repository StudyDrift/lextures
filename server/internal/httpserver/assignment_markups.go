package httpserver

import (
	"encoding/json"
	"net/http"
)

// handleListAssignmentMarkups is GET /api/v1/courses/{course_code}/assignments/{item_id}/markups.
// Reader markups persistence is not ported yet in server; return a schema-compatible empty list.
func (d Deps) handleListAssignmentMarkups() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := d.requireCourseAccess(w, r); !ok {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"markups": []any{}})
	}
}

