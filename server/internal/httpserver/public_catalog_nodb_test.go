package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

// Feature off: public catalog endpoints must return 404 (rollback path), not 405.
func TestPublicCatalog_FeatureOff_Returns404(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFPublicCatalog: false}})
	for _, path := range []string{
		"/api/v1/public/catalog/courses",
		"/api/v1/public/catalog/categories",
		"/api/v1/public/catalog/courses/some-slug",
		"/api/v1/internal/catalog/courses/some-slug/json-ld",
	} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s: got %d, want 404 (body %s)", path, rr.Code, rr.Body.String())
		}
	}
}

// Public catalog list must not require authentication: an anonymous request with
// the feature on validates params (here, 400) rather than redirecting/401ing.
func TestPublicCatalog_NoAuthRequired_ValidatesParams(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFPublicCatalog: true}})
	cases := map[string]string{
		"/api/v1/public/catalog/courses?level=expert":   "invalid level",
		"/api/v1/public/catalog/courses?sort=bogus":     "invalid sort",
		"/api/v1/public/catalog/courses?price_max=-1":   "invalid price",
		"/api/v1/public/catalog/courses?cursor=not!b64": "invalid cursor",
		"/api/v1/public/catalog/courses?limit=0":        "invalid limit",
	}
	for path, name := range cases {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s (%s): got %d, want 400 (body %s)", path, name, rr.Code, rr.Body.String())
		}
	}
}
