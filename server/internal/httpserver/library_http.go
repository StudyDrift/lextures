package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repoLibrary "github.com/lextures/lextures/server/internal/repos/library"
	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
)

// handleOrgLibraryCollection is GET+POST /api/v1/orgs/{orgId}/library.
func (d Deps) handleOrgLibraryCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Config.FFLibrary {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Library feature is not enabled.")
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}

		switch r.Method {
		case http.MethodGet:
			actor, ok := d.meUserID(w, r)
			if !ok {
				return
			}
			_ = actor // any authenticated org member may browse the catalog
			f := repoLibrary.ListBooksFilter{}
			if v := strings.TrimSpace(r.URL.Query().Get("lexile_min")); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					f.LexileMin = &n
				}
			}
			if v := strings.TrimSpace(r.URL.Query().Get("lexile_max")); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 {
					f.LexileMax = &n
				}
			}
			if v := strings.TrimSpace(r.URL.Query().Get("grade_band")); v != "" {
				f.GradeBand = &v
			}
			books, err := repoLibrary.ListBooks(r.Context(), d.Pool, orgID, f)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list library books.")
				return
			}
			out := make([]map[string]any, 0, len(books))
			for i := range books {
				out = append(out, bookToJSON(&books[i]))
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"books": out})

		case http.MethodPost:
			if _, ok := d.orgRoleAccess(w, r, orgID, false); !ok {
				return
			}
			var body struct {
				Title       string  `json:"title"`
				Author      *string `json:"author"`
				ISBN        *string `json:"isbn"`
				CoverURL    *string `json:"coverUrl"`
				LexileLevel *int    `json:"lexileLevel"`
				FPBand      *string `json:"fpBand"`
				GradeBand   *string `json:"gradeBand"`
				Summary     *string `json:"summary"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			title := strings.TrimSpace(body.Title)
			if title == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "title is required.")
				return
			}
			book, err := repoLibrary.CreateBook(r.Context(), d.Pool, repoLibrary.CreateBookParams{
				OrgID:       orgID,
				Title:       title,
				Author:      nilIfEmpty(body.Author),
				ISBN:        nilIfEmpty(body.ISBN),
				CoverURL:    nilIfEmpty(body.CoverURL),
				LexileLevel: body.LexileLevel,
				FPBand:      nilIfEmpty(body.FPBand),
				GradeBand:   nilIfEmpty(body.GradeBand),
				Summary:     nilIfEmpty(body.Summary),
			})
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create library book.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"book": bookToJSON(book)})

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleOrgLibraryItem is GET+DELETE /api/v1/orgs/{orgId}/library/{bookId}.
func (d Deps) handleOrgLibraryItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Config.FFLibrary {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Library feature is not enabled.")
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		bookID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "bookId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid book id.")
			return
		}

		switch r.Method {
		case http.MethodGet:
			if _, ok := d.meUserID(w, r); !ok {
				return
			}
			book, err := repoLibrary.GetBook(r.Context(), d.Pool, orgID, bookID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to get library book.")
				return
			}
			if book == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Book not found.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"book": bookToJSON(book)})

		case http.MethodDelete:
			if _, ok := d.orgRoleAccess(w, r, orgID, false); !ok {
				return
			}
			if err := repoLibrary.DeleteBook(r.Context(), d.Pool, orgID, bookID); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete library book.")
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleMeReadingLog is GET+POST /api/v1/me/reading-log.
func (d Deps) handleMeReadingLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Config.FFLibrary {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Library feature is not enabled.")
			return
		}
		studentID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			limit := 100
			if l := strings.TrimSpace(r.URL.Query().Get("limit")); l != "" {
				if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
					limit = n
				}
			}
			entries, err := repoLibrary.ListReadingLogEntries(r.Context(), d.Pool, studentID, limit)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list reading log.")
				return
			}
			out := make([]map[string]any, 0, len(entries))
			for i := range entries {
				out = append(out, entryToJSON(&entries[i]))
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})

		case http.MethodPost:
			var body struct {
				BookID     *string `json:"bookId"`
				BookTitle  *string `json:"bookTitle"`
				LogDate    string  `json:"logDate"`
				PagesRead  *int    `json:"pagesRead"`
				Reflection *string `json:"reflection"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			dateStr := strings.TrimSpace(body.LogDate)
			if dateStr == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "logDate is required (YYYY-MM-DD).")
				return
			}
			logDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "logDate must be YYYY-MM-DD.")
				return
			}
			if body.BookTitle == nil || strings.TrimSpace(*body.BookTitle) == "" {
				if body.BookID == nil || strings.TrimSpace(*body.BookID) == "" {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "bookTitle or bookId is required.")
					return
				}
			}
			var bookUUID *uuid.UUID
			if body.BookID != nil && strings.TrimSpace(*body.BookID) != "" {
				parsed, err := uuid.Parse(strings.TrimSpace(*body.BookID))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid bookId.")
					return
				}
				bookUUID = &parsed
			}
			bookTitle := nilIfEmpty(body.BookTitle)
			entry, err := repoLibrary.CreateReadingLogEntry(r.Context(), d.Pool,
				studentID, bookUUID, bookTitle, logDate, body.PagesRead, nilIfEmpty(body.Reflection))
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save reading log entry.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"entry": entryToJSON(entry)})

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleCourseReadingDashboard is GET /api/v1/courses/{courseCode}/reading-dashboard.
func (d Deps) handleCourseReadingDashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Config.FFLibrary {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Library feature is not enabled.")
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		courseCode := strings.TrimSpace(chi.URLParam(r, "courseCode"))
		if courseCode == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Course code is required.")
			return
		}
		courseID, err := repoCourse.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to look up course.")
			return
		}
		if courseID == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rows, err := repoLibrary.ReadingDashboard(r.Context(), d.Pool, *courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load reading dashboard.")
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			m := map[string]any{
				"studentId":    row.StudentID.String(),
				"email":        row.Email,
				"weeklyPages":  row.WeeklyPages,
				"totalEntries": row.TotalEntries,
				"totalPages":   row.TotalPages,
			}
			if row.DisplayName != nil {
				m["displayName"] = *row.DisplayName
			} else {
				m["displayName"] = nil
			}
			out = append(out, m)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"students": out})
	}
}

func (d Deps) registerLibraryRoutes(r chi.Router) {
	r.Method(http.MethodGet, "/api/v1/orgs/{orgId}/library", d.handleOrgLibraryCollection())
	r.Method(http.MethodPost, "/api/v1/orgs/{orgId}/library", d.handleOrgLibraryCollection())
	r.Method(http.MethodGet, "/api/v1/orgs/{orgId}/library/{bookId}", d.handleOrgLibraryItem())
	r.Method(http.MethodDelete, "/api/v1/orgs/{orgId}/library/{bookId}", d.handleOrgLibraryItem())
	r.Method(http.MethodGet, "/api/v1/me/reading-log", d.handleMeReadingLog())
	r.Method(http.MethodPost, "/api/v1/me/reading-log", d.handleMeReadingLog())
	r.Method(http.MethodGet, "/api/v1/courses/{courseCode}/reading-dashboard", d.handleCourseReadingDashboard())
}

// ─── JSON helpers ─────────────────────────────────────────────────────────────

func bookToJSON(b *repoLibrary.Book) map[string]any {
	if b == nil {
		return nil
	}
	m := map[string]any{
		"id":        b.ID.String(),
		"orgId":     b.OrgID.String(),
		"title":     b.Title,
		"createdAt": b.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		"updatedAt": b.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if b.Author != nil {
		m["author"] = *b.Author
	} else {
		m["author"] = nil
	}
	if b.ISBN != nil {
		m["isbn"] = *b.ISBN
	} else {
		m["isbn"] = nil
	}
	if b.CoverURL != nil {
		m["coverUrl"] = *b.CoverURL
	} else {
		m["coverUrl"] = nil
	}
	if b.LexileLevel != nil {
		m["lexileLevel"] = *b.LexileLevel
	} else {
		m["lexileLevel"] = nil
	}
	if b.FPBand != nil {
		m["fpBand"] = *b.FPBand
	} else {
		m["fpBand"] = nil
	}
	if b.GradeBand != nil {
		m["gradeBand"] = *b.GradeBand
	} else {
		m["gradeBand"] = nil
	}
	if b.Summary != nil {
		m["summary"] = *b.Summary
	} else {
		m["summary"] = nil
	}
	return m
}

func entryToJSON(e *repoLibrary.ReadingLogEntry) map[string]any {
	if e == nil {
		return nil
	}
	m := map[string]any{
		"id":        e.ID.String(),
		"studentId": e.StudentID.String(),
		"logDate":   e.LogDate.Format("2006-01-02"),
		"loggedAt":  e.LoggedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if e.BookID != nil {
		m["bookId"] = e.BookID.String()
	} else {
		m["bookId"] = nil
	}
	if e.BookTitle != nil {
		m["bookTitle"] = *e.BookTitle
	} else {
		m["bookTitle"] = nil
	}
	if e.PagesRead != nil {
		m["pagesRead"] = *e.PagesRead
	} else {
		m["pagesRead"] = nil
	}
	if e.Reflection != nil {
		m["reflection"] = *e.Reflection
	} else {
		m["reflection"] = nil
	}
	return m
}

func nilIfEmpty(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
