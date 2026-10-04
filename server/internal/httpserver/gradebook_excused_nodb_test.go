package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestGradebookCellExcused_Registered(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/COURSE01/gradebook/cells/00000000-0000-0000-0000-000000000001/excused", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("route not registered: %s", rr.Body.String())
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d: %s", rr.Code, rr.Body.String())
	}
}
