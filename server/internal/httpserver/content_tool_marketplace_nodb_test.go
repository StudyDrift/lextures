package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestContentToolMarketplaceDisabledReturns501(t *testing.T) {
	d := Deps{Config: config.Config{FFContentToolMarketplace: false}}
	h := NewHandler(d)
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/tool-marketplace/tools"},
		{http.MethodGet, "/api/v1/developer/tools"},
		{http.MethodPost, "/api/v1/developer/tools"},
		{http.MethodGet, "/api/v1/admin/tool-reviews"},
	}
	for _, p := range paths {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(p.method, p.path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotImplemented {
			t.Fatalf("%s %s: want 501, got %d body=%s", p.method, p.path, rr.Code, rr.Body.String())
		}
	}
}

func TestContentToolMarketplaceEnabledRequiresAuth(t *testing.T) {
	d := Deps{Config: config.Config{FFContentToolMarketplace: true}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/developer/tools", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}
