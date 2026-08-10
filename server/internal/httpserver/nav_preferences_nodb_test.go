package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNavPreferences_RequireAuth(t *testing.T) {
	d := Deps{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nav/preferences?scope=global", nil)
	rec := httptest.NewRecorder()
	d.handleGetNavPreferences()(rec, req)
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		// meUserID may write 401
		if rec.Code < 400 {
			t.Fatalf("expected auth failure, got %d", rec.Code)
		}
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/nav/preferences", nil)
	rec = httptest.NewRecorder()
	d.handlePutNavPreferences()(rec, req)
	if rec.Code < 400 {
		t.Fatalf("expected auth failure on PUT, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/nav/preferences?scope=global", nil)
	rec = httptest.NewRecorder()
	d.handleDeleteNavPreferences()(rec, req)
	if rec.Code < 400 {
		t.Fatalf("expected auth failure on DELETE, got %d", rec.Code)
	}
}
