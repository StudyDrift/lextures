package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestDiplomas_FeatureOff(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFDiplomas: false}})
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/credentials/templates"},
		{http.MethodPost, "/api/v1/admin/credentials/templates"},
		{http.MethodPost, "/api/v1/admin/credentials/issue"},
		{http.MethodPost, "/api/v1/admin/credentials/issue/batch"},
		{http.MethodGet, "/api/v1/me/diplomas"},
	}
	for _, p := range paths {
		req := httptest.NewRequest(p.method, p.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s: want 401 or 404 got %d body=%s", p.method, p.path, rec.Code, rec.Body.String())
		}
	}
}

func TestDiplomas_MeList_FeatureOn_Unauthenticated(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFDiplomas: true}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/diplomas", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
}

func TestVerify_DiplomasFlagAloneEnablesVerifyRoute(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFDiplomas: true}, Pool: nil})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/verify/some-token", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound && strings.Contains(rec.Body.String(), "not enabled") {
		t.Fatalf("verify should be available when FFDiplomas on; got %d %s", rec.Code, rec.Body.String())
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 (no pool) got %d body=%s", rec.Code, rec.Body.String())
	}
}
