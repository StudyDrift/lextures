package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestMarketingContentFeatureOffReturns404(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFMarketingContent: false}})
	for _, path := range []string{"/api/v1/admin/marketing/articles", "/api/v1/admin/marketing/articles/00000000-0000-0000-0000-000000000000/translations", "/api/v1/admin/marketing/locales", "/api/v1/admin/marketing/categories", "/api/v1/admin/marketing/authors", "/api/v1/admin/marketing/tags", "/api/v1/admin/marketing/redirects", "/api/v1/admin/marketing/media", "/api/v1/public/content/index", "/api/v1/public/content/articles", "/api/v1/public/content/articles/blog/example", "/api/v1/public/content/articles/docs/category/example", "/api/v1/public/content/categories", "/api/v1/public/content/authors", "/api/v1/public/content/authors/example", "/api/v1/public/content/redirects", "/api/v1/public/content/search?q=test", "/api/v1/public/content/media/00000000-0000-0000-0000-000000000000/original.png"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("%s: got %d body=%s", path, rr.Code, rr.Body.String())
		}
	}
}
