package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestCourseTranslationDisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{TranslationMemoryEnabled: false}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-1/translations?target_locale=es", nil)
	w := httptest.NewRecorder()
	d.handleListCourseTranslations()(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
}

func TestQueryTranslationMemoryRequiresAuth(t *testing.T) {
	d := Deps{Config: config.Config{TranslationMemoryEnabled: true}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/translation-memory?course_code=C-1&target_locale=es&text=hello", nil)
	w := httptest.NewRecorder()
	d.handleQueryTranslationMemory()(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", w.Code)
	}
}

func TestPatchContentLocaleInvalidBody(t *testing.T) {
	d := Deps{Config: config.Config{TranslationMemoryEnabled: true}}
	body, _ := json.Marshal(map[string]string{"contentLocale": "not-a-locale"})
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/C-1/me/content-locale", bytes.NewReader(body))
	w := httptest.NewRecorder()
	d.handlePatchMyContentLocale()(w, r)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
		t.Fatalf("expected 401 or 400, got %d", w.Code)
	}
}
