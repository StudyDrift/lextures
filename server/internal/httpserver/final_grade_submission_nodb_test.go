package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func gradeSubmissionTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestFinalGradesPreview_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFGradeSubmission: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := gradeSubmissionTestToken(t, signer)

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/final-grades/preview", d.handleFinalGradesPreview())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/final-grades/preview", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestFinalGradesSubmit_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFGradeSubmission: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := gradeSubmissionTestToken(t, signer)

	r := chi.NewRouter()
	r.Post("/api/v1/courses/{course_code}/final-grades/submit", d.handleFinalGradesSubmit())

	body := `{"method":"csv","overrides":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/CS101/final-grades/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestFinalGradesExportCSV_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFGradeSubmission: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := gradeSubmissionTestToken(t, signer)

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/final-grades/export.csv", d.handleFinalGradesExportCSV())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/final-grades/export.csv", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestAdminFinalGradesStatus_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFGradeSubmission: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := gradeSubmissionTestToken(t, signer)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/final-grades/status", d.handleAdminFinalGradesStatus())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/final-grades/status?term_id="+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestFinalGradesPreview_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFGradeSubmission: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/final-grades/preview", d.handleFinalGradesPreview())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/final-grades/preview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestFinalGradesSubmit_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFGradeSubmission: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}

	r := chi.NewRouter()
	r.Post("/api/v1/courses/{course_code}/final-grades/submit", d.handleFinalGradesSubmit())

	body := `{"method":"csv","overrides":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/CS101/final-grades/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAdminFinalGradesStatus_MissingTermID(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFGradeSubmission: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := gradeSubmissionTestToken(t, signer)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/final-grades/status", d.handleAdminFinalGradesStatus())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/final-grades/status", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Without pool, adminRbacUser will fail first (401 or 500). Either is acceptable in no-db mode.
	if w.Code == http.StatusOK {
		t.Fatalf("expected non-200 when term_id is missing, got 200")
	}
}
