package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestGetQuizProctoringConfig_FeatureDisabled(t *testing.T) {
	cfg := config.Config{FFProctoringIntegration: false}
	d := Deps{Pool: nil, Config: cfg}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/quizzes/{item_id}/proctoring-config", d.handleGetQuizProctoringConfig())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/quizzes/item1/proctoring-config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestPostQuizProctoringConfig_FeatureDisabled(t *testing.T) {
	cfg := config.Config{FFProctoringIntegration: false}
	d := Deps{Pool: nil, Config: cfg}

	r := chi.NewRouter()
	r.Post("/api/v1/courses/{course_code}/quizzes/{item_id}/proctoring-config", d.handlePostQuizProctoringConfig())

	body := `{"externalToolId":"00000000-0000-0000-0000-000000000001","vendor":"honorlock","required":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/CS101/quizzes/item1/proctoring-config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestDeleteQuizProctoringConfig_FeatureDisabled(t *testing.T) {
	cfg := config.Config{FFProctoringIntegration: false}
	d := Deps{Pool: nil, Config: cfg}

	r := chi.NewRouter()
	r.Delete("/api/v1/courses/{course_code}/quizzes/{item_id}/proctoring-config", d.handleDeleteQuizProctoringConfig())

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/CS101/quizzes/item1/proctoring-config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestGetQuizProctoringSession_FeatureDisabled(t *testing.T) {
	cfg := config.Config{FFProctoringIntegration: false}
	d := Deps{Pool: nil, Config: cfg}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/proctoring-session", d.handleGetQuizProctoringSession())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/quizzes/item1/attempts/attempt1/proctoring-session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestProctoringCallback_FeatureDisabled(t *testing.T) {
	cfg := config.Config{FFProctoringIntegration: false}
	d := Deps{Pool: nil, Config: cfg}

	r := chi.NewRouter()
	r.Post("/api/v1/webhooks/proctoring-callback/{vendor}", d.handleProctoringCallback())

	body := `{"attemptId":"00000000-0000-0000-0000-000000000001","vendorSessionId":"sess1","status":"complete","flagCount":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/proctoring-callback/honorlock", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestGetQuizProctoringConfig_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFProctoringIntegration: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/quizzes/{item_id}/proctoring-config", d.handleGetQuizProctoringConfig())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/quizzes/item1/proctoring-config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestProctoringCallback_UnknownVendor(t *testing.T) {
	cfg := config.Config{FFProctoringIntegration: true}
	d := Deps{Pool: nil, Config: cfg}

	r := chi.NewRouter()
	r.Post("/api/v1/webhooks/proctoring-callback/{vendor}", d.handleProctoringCallback())

	body := `{"attemptId":"00000000-0000-0000-0000-000000000001","vendorSessionId":"s","status":"complete","flagCount":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/proctoring-callback/unknownvendor", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
