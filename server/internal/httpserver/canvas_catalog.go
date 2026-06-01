package httpserver

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

type canvasListCoursesBody struct {
	CanvasBaseURL string `json:"canvasBaseUrl"`
	AccessToken   string `json:"accessToken"`
}

type canvasCourseListItem struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	CourseCode    string `json:"courseCode,omitempty"`
	WorkflowState string `json:"workflowState,omitempty"`
	TermName      string `json:"termName,omitempty"`
}

// handleCanvasListCourses is POST /api/v1/integrations/canvas/courses.
// Proxies the Canvas course list for the authenticated user (token is not stored).
func (d Deps) handleCanvasListCourses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		allowed, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, "global:app:course:create")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !allowed {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}

		var body canvasListCoursesBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		canvasBase, err := normalizeCanvasBaseURL(body.CanvasBaseURL, d.effectiveConfig().CanvasAllowedHostSuffixes)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		token := strings.TrimSpace(body.AccessToken)
		if token == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Canvas access token is required.")
			return
		}

		client := canvasHTTPClient()
		q := url.Values{}
		q.Set("enrollment_type", "teacher")
		q.Set("enrollment_state", "active")
		q.Add("state[]", "available")
		q.Add("state[]", "unpublished")
		q.Add("state[]", "completed")
		q.Add("include[]", "term")
		rows, err := canvasGetArrayPaginated(r.Context(), client, canvasBase, token, "courses", q)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInvalidInput, err.Error())
			return
		}

		out := make([]canvasCourseListItem, 0, len(rows))
		for _, row := range rows {
			id := int64At(row, "id")
			if id <= 0 {
				continue
			}
			name := strAt(row, "name", "")
			if name == "" {
				name = strAt(row, "course_code", "Untitled course")
			}
			item := canvasCourseListItem{
				ID:            id,
				Name:          name,
				CourseCode:    strAt(row, "course_code", ""),
				WorkflowState: strAt(row, "workflow_state", ""),
			}
			if term := objAt(row, "term"); term != nil {
				item.TermName = strAt(term, "name", "")
			}
			out = append(out, item)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Name != out[j].Name {
				return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
			}
			return out[i].ID < out[j].ID
		})

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"courses": out})
	}
}
