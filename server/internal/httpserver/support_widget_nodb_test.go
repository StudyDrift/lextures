package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
)

func TestHelpContextualArticles_Unauthorized(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/help/contextual-articles?route=/courses/abc", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHelpContextualArticles_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	s := auth.NewJWTSigner("test-secret")
	h := NewHandler(Deps{JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/help/contextual-articles", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestSupportWidgetGet_BadOrgID(t *testing.T) {
	t.Parallel()
	s := auth.NewJWTSigner("test-secret")
	h := NewHandler(Deps{JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/not-a-uuid/settings/support-widget", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestSupportWidgetPut_BadOrgID(t *testing.T) {
	t.Parallel()
	s := auth.NewJWTSigner("test-secret")
	h := NewHandler(Deps{JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/orgs/not-a-uuid/settings/support-widget", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestToWidgetJSON_NilRow(t *testing.T) {
	t.Parallel()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	m := toWidgetJSON(orgID, nil)
	if m["enabled"] != true {
		t.Fatalf("expected enabled=true for nil row, got %v", m["enabled"])
	}
	if m["provider"] != "crisp" {
		t.Fatalf("expected provider=crisp for nil row, got %v", m["provider"])
	}
}

func TestValidateWidgetBody_ValidProviders(t *testing.T) {
	t.Parallel()
	for _, p := range []string{"crisp", "intercom", "none"} {
		p := p
		body := supportWidgetPutBody{Provider: &p}
		if err := validateWidgetBody(body); err != nil {
			t.Fatalf("unexpected error for provider %q: %v", p, err)
		}
	}
}

func TestValidateWidgetBody_InvalidProvider(t *testing.T) {
	t.Parallel()
	bad := "slack"
	body := supportWidgetPutBody{Provider: &bad}
	if err := validateWidgetBody(body); err == nil {
		t.Fatal("expected error for invalid provider")
	}
}

func TestHelpContextualArticles_RouteRegistered(t *testing.T) {
	t.Parallel()
	s := auth.NewJWTSigner("test-secret")
	h := NewHandler(Deps{JWTSigner: s})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/help/contextual-articles?route=/courses/abc", nil)
	h.ServeHTTP(rr, req)
	// Without a valid token we get 401; 404 means the route was not registered.
	if rr.Code == http.StatusNotFound {
		t.Fatalf("route not registered (404)")
	}
}
