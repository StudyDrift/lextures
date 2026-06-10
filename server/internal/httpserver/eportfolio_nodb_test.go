package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func eportfolioTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

const (
	eportfolioUUID  = "00000000-0000-0000-0000-000000000002"
	eportfolioUUID2 = "00000000-0000-0000-0000-000000000003"
)

// Authenticated owner/reviewer routes (call meUserID after the feature gate).
var eportfolioAuthRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/v1/me/portfolios"},
	{http.MethodPost, "/api/v1/me/portfolios"},
	{http.MethodGet, "/api/v1/me/portfolios/" + eportfolioUUID},
	{http.MethodPatch, "/api/v1/me/portfolios/" + eportfolioUUID},
	{http.MethodDelete, "/api/v1/me/portfolios/" + eportfolioUUID},
	{http.MethodPost, "/api/v1/me/portfolios/" + eportfolioUUID + "/artifacts"},
	{http.MethodPatch, "/api/v1/me/portfolios/" + eportfolioUUID + "/artifacts/" + eportfolioUUID2},
	{http.MethodDelete, "/api/v1/me/portfolios/" + eportfolioUUID + "/artifacts/" + eportfolioUUID2},
	{http.MethodPost, "/api/v1/portfolios/" + eportfolioUUID + "/artifacts/" + eportfolioUUID2 + "/evaluate"},
}

// Routes that are feature-gated but not part of the owner-auth set.
var eportfolioOtherRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/v1/portfolios/some-slug"},
	{http.MethodGet, "/api/v1/admin/programs/" + eportfolioUUID + "/portfolio-outcomes-report"},
}

func TestEportfolioRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFEportfolio: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := eportfolioTestToken(t, signer)

	allRoutes := append(append([]struct {
		method string
		path   string
	}{}, eportfolioAuthRoutes...), eportfolioOtherRoutes...)
	for _, c := range allRoutes {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusNotFound {
				t.Fatalf("expected route to be registered, got 404 for %s %s: %s",
					c.method, c.path, rr.Body.String())
			}
		})
	}
}

func TestEportfolioRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFEportfolio: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	for _, c := range eportfolioAuthRoutes {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 without auth, got %d for %s %s",
					rr.Code, c.method, c.path)
			}
		})
	}
}

func TestEportfolioRoutes_FeatureOff_Returns501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFEportfolio: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := eportfolioTestToken(t, signer)

	allRoutes := append(append([]struct {
		method string
		path   string
	}{}, eportfolioAuthRoutes...), eportfolioOtherRoutes...)
	for _, c := range allRoutes {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusNotImplemented {
				t.Fatalf("expected 501 when feature off, got %d for %s %s: %s",
					rr.Code, c.method, c.path, rr.Body.String())
			}
		})
	}
}

// TestEportfolioPublicRoute_NoAuthRequired confirms the public read endpoint does
// not require authentication when the feature is enabled (it must not 401).
func TestEportfolioPublicRoute_NoAuthRequired(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFEportfolio: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/portfolios/some-slug", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		t.Fatalf("public portfolio route must not require auth, got 401")
	}
}

func TestParseUUIDList(t *testing.T) {
	ids, err := parseUUIDList([]string{eportfolioUUID, "", "  " + eportfolioUUID2 + "  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}
	if _, err := parseUUIDList([]string{"not-a-uuid"}); err == nil {
		t.Fatalf("expected error for malformed uuid")
	}
}
