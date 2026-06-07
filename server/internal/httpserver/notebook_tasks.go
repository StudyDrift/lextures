package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/notebooktasks"
)

func (d Deps) registerNotebookTaskRoutes(r chi.Router) {
	r.Get("/api/v1/me/notebook-tasks", d.handleListNotebookTasks())
	r.Post("/api/v1/me/notebook-tasks", d.handleUpsertNotebookTask())
	r.Patch("/api/v1/me/notebook-tasks/{id}", d.handlePatchNotebookTask())
}

type notebookTaskJSON struct {
	ID             string  `json:"id"`
	CourseCode     string  `json:"courseCode"`
	NotebookPageID string  `json:"notebookPageId"`
	TaskText       string  `json:"taskText"`
	Completed      bool    `json:"completed"`
	DueAt          *string `json:"dueAt"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

func notebookTaskToJSON(t repo.Task) notebookTaskJSON {
	out := notebookTaskJSON{
		ID:             t.ID.String(),
		CourseCode:     t.CourseCode,
		NotebookPageID: t.NotebookPageID,
		TaskText:       t.TaskText,
		Completed:      t.Completed,
		CreatedAt:      t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      t.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if t.DueAt != nil {
		s := t.DueAt.UTC().Format(time.RFC3339)
		out.DueAt = &s
	}
	return out
}

func (d Deps) handleListNotebookTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database not available.")
			return
		}
		tasks, err := repo.ListOpen(r.Context(), d.Pool, userID, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load notebook tasks.")
			return
		}
		out := make([]notebookTaskJSON, 0, len(tasks))
		for _, t := range tasks {
			out = append(out, notebookTaskToJSON(t))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"tasks": out})
	}
}

type upsertNotebookTaskBody struct {
	ID             string  `json:"id"`
	CourseCode     string  `json:"courseCode"`
	NotebookPageID string  `json:"notebookPageId"`
	TaskText       string  `json:"taskText"`
	Completed      *bool   `json:"completed"`
	DueAt          *string `json:"dueAt"`
}

func (d Deps) handleUpsertNotebookTask() http.HandlerFunc {
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
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database not available.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body upsertNotebookTaskBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		taskID, err := uuid.Parse(strings.TrimSpace(body.ID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid task id.")
			return
		}
		courseCode := strings.TrimSpace(body.CourseCode)
		pageID := strings.TrimSpace(body.NotebookPageID)
		if courseCode == "" || pageID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseCode and notebookPageId are required.")
			return
		}
		text := strings.TrimSpace(body.TaskText)
		if utf8.RuneCountInString(text) > 2000 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Task text is too long.")
			return
		}
		completed := false
		if body.Completed != nil {
			completed = *body.Completed
		}
		var dueAt *time.Time
		if body.DueAt != nil && strings.TrimSpace(*body.DueAt) != "" {
			parsed, perr := time.Parse(time.RFC3339, strings.TrimSpace(*body.DueAt))
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dueAt.")
				return
			}
			dueAt = &parsed
		}
		t := repo.Task{
			ID:             taskID,
			UserID:         userID,
			CourseCode:     courseCode,
			NotebookPageID: pageID,
			TaskText:       text,
			Completed:      completed,
			DueAt:          dueAt,
		}
		if err := repo.Upsert(r.Context(), d.Pool, userID, t); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save notebook task.")
			return
		}
		saved, err := repo.Get(r.Context(), d.Pool, userID, taskID)
		if err != nil || saved == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load saved task.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(notebookTaskToJSON(*saved))
	}
}

type patchNotebookTaskBody struct {
	TaskText  *string `json:"taskText"`
	Completed *bool   `json:"completed"`
	DueAt     *string `json:"dueAt"`
	ClearDue  *bool   `json:"clearDue"`
}

func (d Deps) handlePatchNotebookTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database not available.")
			return
		}
		rawID := chi.URLParam(r, "id")
		taskID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid task id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body patchNotebookTaskBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		var text *string
		if body.TaskText != nil {
			t := strings.TrimSpace(*body.TaskText)
			if utf8.RuneCountInString(t) > 2000 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Task text is too long.")
				return
			}
			text = &t
		}
		var dueAt *time.Time
		clearDue := body.ClearDue != nil && *body.ClearDue
		if !clearDue && body.DueAt != nil && strings.TrimSpace(*body.DueAt) != "" {
			parsed, perr := time.Parse(time.RFC3339, strings.TrimSpace(*body.DueAt))
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dueAt.")
				return
			}
			dueAt = &parsed
		}
		updated, err := repo.Patch(r.Context(), d.Pool, userID, taskID, text, body.Completed, dueAt, clearDue)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update notebook task.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Task not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(notebookTaskToJSON(*updated))
	}
}
