package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestSettingsLocaleGet_RouteRegistered(t *testing.T) {
	d := Deps{JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/settings/locale", nil)
	h.ServeHTTP(rr, r)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("expected /api/v1/settings/locale to be registered, got 404")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without bearer token, got %d", rr.Code)
	}
}

func TestSettingsLocalePut_RouteRegistered(t *testing.T) {
	d := Deps{JWTSigner: auth.NewJWTSigner("01234567890123456789012345678901"), Config: config.Config{}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/settings/locale", nil)
	h.ServeHTTP(rr, r)
	if rr.Code == http.StatusNotFound {
		t.Fatalf("expected PUT /api/v1/settings/locale to be registered, got 404")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without bearer token, got %d", rr.Code)
	}
}

func TestNormalizeLocaleInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in    string
		ok    bool
		out   string
	}{
		{"en", true, "en"},
		{"es", true, "es"},
		{"fr-CA", true, "fr-CA"},
		{"EN", false, ""},
		{"de", false, ""},
		{"", false, ""},
		{"en-us", false, ""},
	}
	for _, tc := range cases {
		got, err := normalizeLocaleInput(tc.in)
		if tc.ok && err != nil {
			t.Errorf("%q: unexpected error %v", tc.in, err)
			continue
		}
		if !tc.ok && err == nil {
			t.Errorf("%q: expected error", tc.in)
			continue
		}
		if tc.ok && got != tc.out {
			t.Errorf("%q: got %q want %q", tc.in, got, tc.out)
		}
	}
}
