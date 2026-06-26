package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/studenttodos"
)

func (d Deps) registerStudentTodoRoutes(r chi.Router) {
	r.Get("/api/v1/me/student-todo-board", d.handleGetStudentTodoBoard())
	r.Put("/api/v1/me/student-todo-board", d.handlePutStudentTodoBoard())
}

type studentTodoPlacementJSON struct {
	ItemKey   string `json:"itemKey"`
	ColumnID  string `json:"columnId"`
	SortOrder int    `json:"sortOrder"`
}

func (d Deps) handleGetStudentTodoBoard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database not available.")
			return
		}
		placements, err := repo.ListPlacements(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load todo board.")
			return
		}
		out := make([]studentTodoPlacementJSON, 0, len(placements))
		for _, p := range placements {
			out = append(out, studentTodoPlacementJSON{
				ItemKey:   p.ItemKey,
				ColumnID:  p.ColumnID,
				SortOrder: p.SortOrder,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"placements": out})
	}
}

type putStudentTodoBoardBody struct {
	Columns map[string][]string `json:"columns"`
}

func (d Deps) handlePutStudentTodoBoard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
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
		var body putStudentTodoBoardBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.Columns == nil {
			body.Columns = map[string][]string{}
		}
		if err := repo.ReplaceBoard(r.Context(), d.Pool, userID, body.Columns); err != nil {
			if strings.Contains(err.Error(), "invalid todo") || strings.Contains(err.Error(), "duplicate") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save todo board.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}