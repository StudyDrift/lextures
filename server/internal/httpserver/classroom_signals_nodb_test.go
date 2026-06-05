package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/classroomsignals"
)

func csTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

var classroomSignalsRoutes = []struct {
	method string
	path   string
}{
	{http.MethodPost, "/api/v1/sections/00000000-0000-0000-0000-000000000001/hall-passes"},
	{http.MethodGet, "/api/v1/sections/00000000-0000-0000-0000-000000000001/hall-passes/active"},
	{http.MethodPatch, "/api/v1/hall-passes/00000000-0000-0000-0000-000000000002"},
	{http.MethodPost, "/api/v1/courses/00000000-0000-0000-0000-000000000003/questions"},
	{http.MethodGet, "/api/v1/courses/00000000-0000-0000-0000-000000000003/questions"},
	{http.MethodPatch, "/api/v1/courses/00000000-0000-0000-0000-000000000003/questions/00000000-0000-0000-0000-000000000004"},
}

func TestClassroomSignalsRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFClassroomSignals: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := csTestToken(t, signer)

	for _, c := range classroomSignalsRoutes {
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

func TestClassroomSignalsRoutes_Unauthenticated401(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFClassroomSignals: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	for _, c := range classroomSignalsRoutes {
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

func TestClassroomSignalsRoutes_FeatureOff_Returns501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFClassroomSignals: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := csTestToken(t, signer)

	for _, c := range classroomSignalsRoutes {
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

func TestClassroomSignalsRoutes_InvalidUUIDs_Returns400(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFClassroomSignals: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := csTestToken(t, signer)

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/sections/not-a-uuid/hall-passes", `{"destination":"bathroom"}`},
		{http.MethodGet, "/api/v1/sections/not-a-uuid/hall-passes/active", ""},
		{http.MethodPatch, "/api/v1/hall-passes/not-a-uuid", `{"status":"approved"}`},
		{http.MethodPost, "/api/v1/courses/not-a-uuid/questions", `{"question":"hi"}`},
		{http.MethodGet, "/api/v1/courses/not-a-uuid/questions", ""},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			var body *bytes.Reader
			if c.body != "" {
				body = bytes.NewReader([]byte(c.body))
			}
			var req *http.Request
			if body == nil {
				req = httptest.NewRequest(c.method, c.path, nil)
			} else {
				req = httptest.NewRequest(c.method, c.path, body)
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d for %s %s: %s",
					rr.Code, c.method, c.path, rr.Body.String())
			}
		})
	}
}

func TestHallPassStateMachine(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{classroomsignals.StatusRequested, classroomsignals.StatusApproved, true},
		{classroomsignals.StatusRequested, classroomsignals.StatusDenied, true},
		{classroomsignals.StatusRequested, classroomsignals.StatusReturned, false},
		{classroomsignals.StatusApproved, classroomsignals.StatusReturned, true},
		{classroomsignals.StatusApproved, classroomsignals.StatusDenied, false},
		{classroomsignals.StatusApproved, classroomsignals.StatusRequested, false},
		{classroomsignals.StatusReturned, classroomsignals.StatusApproved, false},
		{classroomsignals.StatusDenied, classroomsignals.StatusApproved, false},
	}
	for _, c := range cases {
		if got := classroomsignals.CanTransition(c.from, c.to); got != c.want {
			t.Errorf("CanTransition(%q,%q) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestAllowedDestinations(t *testing.T) {
	for _, d := range []string{"bathroom", "office", "library", "nurse", "other"} {
		if !classroomsignals.IsAllowedDestination(d) {
			t.Errorf("expected %q to be allowed", d)
		}
	}
	for _, d := range []string{"", "playground", "lounge", "OFFICE"} {
		if classroomsignals.IsAllowedDestination(d) {
			t.Errorf("expected %q to be rejected", d)
		}
	}
}
