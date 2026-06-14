package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestPutCourseHeroImage_Not404NoRoute(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	rr := httptest.NewRecorder()
	body := `{"imageUrl":"https://example.com/hero.png"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/C-TEST/hero-image", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	tok, _ := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.com", "", "", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("expected handler to be registered, got 404: %s", rr.Body.String())
	}
}