package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TD.5 FR-1/FR-2 — pin observable OPTIONS and wrong-method behaviour for single-method
// routes. These paths go through corsAll + chi MethodNotAllowed (not in-handler dispatch).
//
// Current contract (must remain byte-stable for clients):
//   - OPTIONS → 204, empty body, CORS allow-* headers (corsAll short-circuit)
//   - wrong method on a registered path → 405 JSON METHOD_NOT_ALLOWED (routerMethodNotAllowedHandler)

func TestTD5_OPTIONS_singleMethodRoute(t *testing.T) {
	h := NewHandler(Deps{})
	paths := []string{
		"/api/v1/public/onboarding/track", // POST-only
		"/api/v1/search",                  // GET-only
		"/health",                         // GET-only
		"/api/v1/courses/X/outcomes",      // GET+POST registered; OPTIONS still corsAll
	}
	for _, path := range paths {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodOptions, path, nil)
		r.Header.Set("Origin", "http://localhost:5173")
		r.Header.Set("Access-Control-Request-Method", "POST")
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusNoContent {
			t.Errorf("OPTIONS %s: status=%d want 204", path, rr.Code)
		}
		if rr.Body.Len() != 0 {
			t.Errorf("OPTIONS %s: body should be empty, got %q", path, rr.Body.String())
		}
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("OPTIONS %s: ACAO=%q want *", path, got)
		}
		if got := rr.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
			t.Errorf("OPTIONS %s: ACAM=%q missing POST", path, got)
		}
	}
}

func TestTD5_WrongMethod_singleMethodRoute(t *testing.T) {
	h := NewHandler(Deps{})
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/public/onboarding/track"},
		{http.MethodDelete, "/api/v1/public/onboarding/track"},
		{http.MethodPost, "/api/v1/search"},
		{http.MethodPost, "/health"},
		{http.MethodPut, "/api/v1/courses/X/outcomes"},
	}
	for _, c := range cases {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(c.method, c.path, nil)
		r.Header.Set("Origin", "http://localhost:5173")
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status=%d want 405 body=%s", c.method, c.path, rr.Code, rr.Body.String())
			continue
		}
		ct := rr.Header().Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			t.Errorf("%s %s: Content-Type=%q want application/json", c.method, c.path, ct)
		}
		var body struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Errorf("%s %s: invalid JSON body: %v (%q)", c.method, c.path, err, rr.Body.String())
			continue
		}
		if body.Error.Code != "METHOD_NOT_ALLOWED" {
			t.Errorf("%s %s: error.code=%q want METHOD_NOT_ALLOWED", c.method, c.path, body.Error.Code)
		}
		if !strings.Contains(body.Error.Message, c.method) || !strings.Contains(body.Error.Message, c.path) {
			t.Errorf("%s %s: message=%q should mention method and path", c.method, c.path, body.Error.Message)
		}
		// CORS still applied by outermost middleware
		if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("%s %s: ACAO=%q want *", c.method, c.path, got)
		}
	}
}

func TestTD5_CORSPreflight_mutatingEndpoint(t *testing.T) {
	// AC-3: browser CORS preflight against a mutating endpoint succeeds as before.
	h := NewHandler(Deps{})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/public/onboarding/track", nil)
	r.Header.Set("Origin", "https://app.example.com")
	r.Header.Set("Access-Control-Request-Method", "POST")
	r.Header.Set("Access-Control-Request-Headers", "content-type,authorization")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d want 204", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("ACAO=%q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	acam := rr.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(acam, "POST") || !strings.Contains(acam, "OPTIONS") {
		t.Fatalf("ACAM=%q", acam)
	}
	if rr.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Fatal("missing Access-Control-Allow-Headers")
	}
}
