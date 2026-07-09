package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestMarketplacePurchase_NoAuth_Returns401(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFCourseMarketplace: true}, JWTSigner: nil})
	for _, path := range []string{
		"/api/v1/marketplace/courses/some-slug/claim",
		"/api/v1/marketplace/courses/some-slug/checkout",
	} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s: got %d, want 401 (body %s)", path, rr.Code, rr.Body.String())
		}
	}
}

func TestMarketplaceCheckoutHint(t *testing.T) {
	if got := marketplaceCheckoutHint("my-slug", "CS101"); got != "/marketplace/my-slug" {
		t.Fatalf("got %q", got)
	}
	if got := marketplaceCheckoutHint("", "CS101"); got != "/marketplace/CS101" {
		t.Fatalf("got %q", got)
	}
	if got := marketplaceCheckoutHint("", ""); got != "/marketplace" {
		t.Fatalf("got %q", got)
	}
}
