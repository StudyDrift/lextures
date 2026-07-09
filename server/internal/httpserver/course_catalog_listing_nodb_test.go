package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestCatalogListingOff_WhenBothDisabled(t *testing.T) {
	d := Deps{Config: config.Config{FFPublicCatalog: false, FFCourseMarketplace: false}}
	rr := httptest.NewRecorder()
	if !d.catalogListingOff(rr) {
		t.Fatal("expected catalogListingOff to report disabled")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", rr.Code)
	}
}

func TestCatalogListingOff_WhenMarketplaceEnabled(t *testing.T) {
	d := Deps{Config: config.Config{FFPublicCatalog: false, FFCourseMarketplace: true}}
	rr := httptest.NewRecorder()
	if d.catalogListingOff(rr) {
		t.Fatal("expected catalog listing available when marketplace on")
	}
}

func TestCatalogListingOff_WhenPublicCatalogEnabled(t *testing.T) {
	d := Deps{Config: config.Config{FFPublicCatalog: true, FFCourseMarketplace: false}}
	rr := httptest.NewRecorder()
	if d.catalogListingOff(rr) {
		t.Fatal("expected catalog listing available when public catalog on")
	}
	if rr.Body.Len() > 0 {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestCatalogListingOff_Message(t *testing.T) {
	d := Deps{Config: config.Config{}}
	rr := httptest.NewRecorder()
	d.catalogListingOff(rr)
	if !strings.Contains(rr.Body.String(), "not enabled") {
		t.Fatalf("body: %s", rr.Body.String())
	}
}
