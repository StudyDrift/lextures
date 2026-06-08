package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestGetSubmissionOriginality_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFPlagiarismChecks: true, OriginalityDetectionEnabled: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/assignments/{item_id}/submissions/{submission_id}/originality", d.handleGetSubmissionOriginality())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/assignments/a/submissions/b/originality", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestGetSubmissionOriginality_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFPlagiarismChecks: false, OriginalityDetectionEnabled: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/assignments/{item_id}/submissions/{submission_id}/originality", d.handleGetSubmissionOriginality())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/assignments/a/submissions/b/originality", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestPostSubmissionOriginalityRetry_FeatureDisabled(t *testing.T) {
	cfg := config.Config{FFPlagiarismChecks: false, OriginalityDetectionEnabled: true}
	d := Deps{Pool: nil, Config: cfg}

	r := chi.NewRouter()
	r.Post("/api/v1/courses/{course_code}/assignments/{item_id}/submissions/{submission_id}/originality/retry", d.handlePostSubmissionOriginalityRetry())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/CS101/assignments/a/submissions/b/originality/retry", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestGetCoursePlagiarismSettings_FeatureDisabled(t *testing.T) {
	cfg := config.Config{FFPlagiarismChecks: false, OriginalityDetectionEnabled: true}
	d := Deps{Pool: nil, Config: cfg}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/plagiarism-settings", d.handleGetCoursePlagiarismSettings())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/plagiarism-settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}
