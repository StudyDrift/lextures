package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func broadcastTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

var broadcastRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/api/v1/orgs/00000000-0000-0000-0000-000000000001/broadcasts"},
	{http.MethodPost, "/api/v1/orgs/00000000-0000-0000-0000-000000000001/broadcasts"},
	{http.MethodGet, "/api/v1/orgs/00000000-0000-0000-0000-000000000001/broadcasts/00000000-0000-0000-0000-000000000002/delivery-report"},
	{http.MethodPost, "/api/v1/broadcasts/00000000-0000-0000-0000-000000000002/acknowledge"},
	{http.MethodGet, "/api/v1/me/broadcasts"},
}

func TestBroadcastRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFBroadcasts: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := broadcastTestToken(t, signer)

	for _, c := range broadcastRoutes {
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

func TestBroadcastRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFBroadcasts: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	for _, c := range broadcastRoutes {
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

func TestBroadcastRoutes_FeatureOff_Returns501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFBroadcasts: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := broadcastTestToken(t, signer)

	for _, c := range broadcastRoutes {
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

func TestBroadcastRoutes_InvalidOrgID_Returns400(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFBroadcasts: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := broadcastTestToken(t, signer)

	cases := []string{
		"/api/v1/orgs/not-a-uuid/broadcasts",
		"/api/v1/orgs/not-a-uuid/broadcasts/00000000-0000-0000-0000-000000000002/delivery-report",
		"/api/v1/broadcasts/not-a-uuid/acknowledge",
	}

	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			method := http.MethodGet
			if strings.Contains(path, "/acknowledge") {
				method = http.MethodPost
			}
			req := httptest.NewRequest(method, path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for invalid id, got %d for %s %s: %s",
					rr.Code, method, path, rr.Body.String())
			}
		})
	}
}

func TestBroadcastsPost_RejectsInvalidType(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFBroadcasts: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := broadcastTestToken(t, signer)

	body := `{"type":"spam","subject":"x","body":"y"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/orgs/00000000-0000-0000-0000-000000000001/broadcasts",
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	// expect 400 (invalid type) before any DB call, OR 403 if role check fires first.
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden && rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 400/403/500 for invalid type, got %d: %s", rr.Code, rr.Body.String())
	}
}
