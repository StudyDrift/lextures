package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestDetectBrowserLocale(t *testing.T) {
	if got := detectBrowserLocale(""); got != "en" {
		t.Fatalf("empty: got %q", got)
	}
	if got := detectBrowserLocale("ar-SA,ar;q=0.9,en;q=0.8"); got != "ar-SA" {
		t.Fatalf("arabic: got %q", got)
	}
	if got := detectBrowserLocale("fr-FR,fr;q=0.9"); got != "fr-FR" {
		t.Fatalf("french: got %q", got)
	}
}

func TestHandleGetPublicLocaleDefaults(t *testing.T) {
	d := Deps{Config: config.Config{}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/public/locale-defaults", nil)
	r.Header.Set("Accept-Language", "he-IL,he;q=0.9")
	w := httptest.NewRecorder()
	d.handleGetPublicLocaleDefaults()(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestNormalizeLocaleInput_rtl(t *testing.T) {
	got, err := normalizeLocaleInput("ar")
	if err != nil || got != "ar" {
		t.Fatalf("got %q err %v", got, err)
	}
	_, err = normalizeLocaleInput("xx")
	if err == nil {
		t.Fatal("expected error for unsupported locale")
	}
}
