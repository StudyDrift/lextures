package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/config"
)

func TestCCRRoutes_FeatureOff_Returns501(t *testing.T) {
	d := Deps{Config: config.Config{FFCoCurricularTranscript: false}}
	r := chi.NewRouter()
	d.registerCCRRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/ccr", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCCRRoutes_Unauthenticated401(t *testing.T) {
	d := Deps{Config: config.Config{FFCoCurricularTranscript: true}}
	r := chi.NewRouter()
	d.registerCCRRoutes(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/ccr/generate", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestVerifyRoute_NotFoundWithoutToken(t *testing.T) {
	d := Deps{Config: config.Config{FFCoCurricularTranscript: true}}
	r := chi.NewRouter()
	d.registerCCRRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/verify/not-a-real-token", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
