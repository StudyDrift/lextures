package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestCalendarFeedRoutes_NoDB_Unauthenticated(t *testing.T) {
	t.Parallel()
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCalendarFeeds: true}
	h := NewHandler(Deps{JWTSigner: signer, Config: cfg})

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/calendar.ics"},
		{http.MethodGet, "/api/v1/me/calendar-token"},
		{http.MethodPost, "/api/v1/me/calendar-token"},
	}
	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d: %s", rt.method, rt.path, rr.Code, rr.Body.String())
		}
	}
}

func TestCalendarFeedRoutes_NoDB_FeatureDisabled(t *testing.T) {
	t.Parallel()
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCalendarFeeds: false}
	h := NewHandler(Deps{JWTSigner: signer, Config: cfg})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/calendar.ics?token=lcf_test", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when feature disabled, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCalDAVWellKnown_NoDB_MissingToken(t *testing.T) {
	t.Parallel()
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCalendarFeeds: true}
	h := NewHandler(Deps{JWTSigner: signer, Config: cfg})

	req := httptest.NewRequest(http.MethodGet, "/.well-known/caldav", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing token, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCalDAVCollection_NoDB_InvalidUser(t *testing.T) {
	t.Parallel()
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCalendarFeeds: true}
	h := NewHandler(Deps{JWTSigner: signer, Config: cfg})

	req := httptest.NewRequest(http.MethodGet, "/caldav/users/not-a-uuid/?token=lcf_test", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid user id, got %d: %s", rr.Code, rr.Body.String())
	}
}

