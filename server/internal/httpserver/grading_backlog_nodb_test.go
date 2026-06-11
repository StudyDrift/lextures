package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/auth"
)

func gradingBacklogTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestCourseGradingBacklog_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Pool: nil, JWTSigner: signer}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/grading-backlog", d.handleCourseGradingBacklog())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/grading-backlog", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestCourseGradingBacklog_NoDatabase(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Pool: nil, JWTSigner: signer}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/grading-backlog", d.handleCourseGradingBacklog())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/grading-backlog", nil)
	req.Header.Set("Authorization", "Bearer "+gradingBacklogTestToken(t, signer))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}