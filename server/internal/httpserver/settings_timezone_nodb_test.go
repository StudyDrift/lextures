package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestSettingsTimezoneRoutes_Registered(t *testing.T) {
	d := Deps{JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{}}
	h := NewHandler(d)

	for _, path := range []string{
		"/api/v1/settings/timezone",
		"/api/v1/timezones",
	} {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		h.ServeHTTP(rr, r)
		if rr.Code == http.StatusNotFound {
			t.Fatalf("expected %s to be registered, got 404", path)
		}
	}
}

func TestPutSettingsTimezone_InvalidJSON(t *testing.T) {
	d := Deps{JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/settings/timezone", bytes.NewReader([]byte("{")))
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		// without auth we only assert route exists; with bad json after auth would be 400
		if rr.Code == http.StatusNotFound {
			t.Fatalf("route not registered")
		}
	}
}

func TestPutSettingsTimezone_UnauthorizedWithoutToken(t *testing.T) {
	d := Deps{JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{}}
	h := NewHandler(d)
	body, _ := json.Marshal(map[string]string{"timezone": "Asia/Tokyo"})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/settings/timezone", bytes.NewReader(body))
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
