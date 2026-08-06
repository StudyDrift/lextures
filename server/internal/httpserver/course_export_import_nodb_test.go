package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCourseExportGet_Unauthorized(t *testing.T) {
	d := Deps{}
	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/export", d.handleCourseExportGet())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST01/export", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestCourseExportGet_MethodNotAllowed(t *testing.T) {
	// TD.5: single-method route — chi returns 405 for the wrong verb (not in-handler).
	d := Deps{}
	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/export", d.handleCourseExportGet())
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-TEST01/export", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want %d body=%s", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
}

func TestCourseImportPost_Unauthorized(t *testing.T) {
	d := Deps{}
	r := chi.NewRouter()
	r.Post("/api/v1/courses/{course_code}/import", d.handleCourseImportPost())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-TEST01/import", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestCourseImportPost_MethodNotAllowed(t *testing.T) {
	// TD.5: single-method route — chi returns 405 for the wrong verb (not in-handler).
	d := Deps{}
	r := chi.NewRouter()
	r.Post("/api/v1/courses/{course_code}/import", d.handleCourseImportPost())
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST01/import", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want %d body=%s", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
}
