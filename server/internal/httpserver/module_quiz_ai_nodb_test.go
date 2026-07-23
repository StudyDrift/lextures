package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestModuleQuizAIRoutes_Registered(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/courses/demo/quizzes/00000000-0000-0000-0000-000000000001/generate-questions"},
		{http.MethodPost, "/api/v1/courses/demo/quizzes/00000000-0000-0000-0000-000000000001/import-questions-markdown"},
		{http.MethodGet, "/api/v1/courses/demo/questions"},
		{http.MethodGet, "/api/v1/courses/demo/questions/00000000-0000-0000-0000-000000000002"},
	}
	for _, c := range paths {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusNotFound {
				t.Fatalf("expected route registered, got 404: %s", rr.Body.String())
			}
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 without auth, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}
