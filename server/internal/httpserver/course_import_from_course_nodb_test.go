package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestPostCourseImportFromCourse_Unauthorized(t *testing.T) {
	d := Deps{}
	r := chi.NewRouter()
	r.Post("/api/v1/courses/import/from-course", d.handlePostCourseImportFromCourse())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/import/from-course", strings.NewReader(`{"sourceCourseCode":"C-TEST01","title":"Copy"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d body=%s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}