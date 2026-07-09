package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestCourseMarketplaceOff_WhenDisabled(t *testing.T) {
	d := Deps{Config: config.Config{FFCourseMarketplace: false}}
	rr := httptest.NewRecorder()
	if !d.courseMarketplaceOff(rr) {
		t.Fatal("expected courseMarketplaceOff to report disabled")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404 (body %s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Marketplace is not enabled") {
		t.Fatalf("body: %s", rr.Body.String())
	}
}

func TestCourseMarketplaceOff_WhenEnabled(t *testing.T) {
	d := Deps{Config: config.Config{FFCourseMarketplace: true}}
	rr := httptest.NewRecorder()
	if d.courseMarketplaceOff(rr) {
		t.Fatal("expected courseMarketplaceOff to allow when enabled")
	}
	if rr.Body.Len() > 0 {
		t.Fatalf("unexpected body written when flag on: %s", rr.Body.String())
	}
}
