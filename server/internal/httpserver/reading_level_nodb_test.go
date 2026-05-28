package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestReadingLevel_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{ReadingLevelEnabled: false}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/demo/items/00000000-0000-0000-0000-000000000001/reading-level", nil)
	rec := httptest.NewRecorder()
	d.handleGetItemReadingLevel()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}
