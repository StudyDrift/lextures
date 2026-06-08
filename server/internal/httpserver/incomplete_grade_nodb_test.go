package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/config"
)

func TestIncompleteGradePostDisabled(t *testing.T) {
	cfg := config.Config{FFIncompleteGradeWorkflow: false}
	d := Deps{Config: cfg}
	r := chi.NewRouter()
	r.Post("/api/v1/courses/{course_code}/enrollments/{enrollment_id}/incomplete", d.handleIncompleteGradePost())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/TEST/enrollments/00000000-0000-0000-0000-000000000001/incomplete", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminIncompletesDisabled(t *testing.T) {
	cfg := config.Config{FFIncompleteGradeWorkflow: false}
	d := Deps{Config: cfg}
	r := chi.NewRouter()
	r.Get("/api/v1/admin/incompletes", d.handleAdminIncompletes())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/incompletes", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d body=%s", rec.Code, rec.Body.String())
	}
}
