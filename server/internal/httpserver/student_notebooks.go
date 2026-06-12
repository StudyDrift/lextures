package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/studentnotebooks"
)

// maxNotebookBytes caps one notebook document (all pages of one course).
const maxNotebookBytes = 2 << 20 // 2 MiB

// The course code travels as a query parameter — a path segment would collide with the
// sibling /api/v1/me/notebooks/query and /notebooks/flashcards routes.
func (d Deps) registerStudentNotebookRoutes(r chi.Router) {
	r.Get("/api/v1/me/notebooks", d.handleListStudentNotebooks())
	r.Put("/api/v1/me/notebooks", d.handlePutStudentNotebook())
}

type studentNotebookJSON struct {
	CourseCode string          `json:"courseCode"`
	UpdatedAt  string          `json:"updatedAt"`
	Data       json.RawMessage `json:"data"`
}

func studentNotebookToJSON(n repo.Notebook) studentNotebookJSON {
	return studentNotebookJSON{
		CourseCode: n.CourseCode,
		UpdatedAt:  n.UpdatedAt.UTC().Format(time.RFC3339),
		Data:       json.RawMessage(n.Data),
	}
}

func (d Deps) handleListStudentNotebooks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database not available.")
			return
		}
		notebooks, err := repo.List(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load notebooks.")
			return
		}
		out := make([]studentNotebookJSON, 0, len(notebooks))
		for _, n := range notebooks {
			out = append(out, studentNotebookToJSON(n))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"notebooks": out})
	}
}

// putStudentNotebookBody mirrors the client CourseNotebookStore (format v2). Pages are kept
// as raw JSON so clients can evolve page fields without server churn; only the envelope is checked.
type putStudentNotebookBody struct {
	FormatVersion int               `json:"formatVersion"`
	UpdatedAt     string            `json:"updatedAt"`
	Pages         []json.RawMessage `json:"pages"`
}

func (d Deps) handlePutStudentNotebook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database not available.")
			return
		}
		courseCode := strings.TrimSpace(r.URL.Query().Get("courseCode"))
		if courseCode == "" || len(courseCode) > 200 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course code.")
			return
		}
		b, err := io.ReadAll(io.LimitReader(r.Body, maxNotebookBytes+1))
		_ = r.Body.Close()
		if err != nil || len(b) > maxNotebookBytes {
			apierr.WriteJSON(w, http.StatusRequestEntityTooLarge, apierr.CodeInvalidInput, "Notebook is too large.")
			return
		}
		var body putStudentNotebookBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.FormatVersion != 2 || len(body.Pages) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Expected a format v2 notebook with at least one page.")
			return
		}
		updatedAt, perr := time.Parse(time.RFC3339, strings.TrimSpace(body.UpdatedAt))
		if perr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid updatedAt.")
			return
		}
		saved, err := repo.Upsert(r.Context(), d.Pool, userID, courseCode, b, updatedAt)
		if err != nil || saved == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save notebook.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(studentNotebookToJSON(*saved))
	}
}
