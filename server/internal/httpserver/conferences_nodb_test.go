package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/conferences"
)

func confTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

var conferenceRoutes = []struct {
	method string
	path   string
}{
	{http.MethodPost, "/api/v1/teachers/00000000-0000-0000-0000-000000000001/conference-availability"},
	{http.MethodGet, "/api/v1/teachers/00000000-0000-0000-0000-000000000001/conference-slots?date=2025-11-18"},
	{http.MethodPost, "/api/v1/conference-slots/00000000-0000-0000-0000-000000000002/book"},
	{http.MethodDelete, "/api/v1/conference-slots/00000000-0000-0000-0000-000000000002/book"},
	{http.MethodGet, "/api/v1/conference-slots/00000000-0000-0000-0000-000000000002/ical"},
	{http.MethodGet, "/api/v1/parent/conference-teachers?studentId=00000000-0000-0000-0000-000000000003"},
	{http.MethodGet, "/api/v1/admin/org-units/00000000-0000-0000-0000-000000000004/conference-schedule?date=2025-11-18"},
}

func TestConferenceRoutes_NotFound404(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFConferenceScheduling: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := confTestToken(t, signer)

	for _, c := range conferenceRoutes {
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

func TestConferenceRoutes_FeatureOff_Returns501(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFConferenceScheduling: false}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := confTestToken(t, signer)

	for _, c := range conferenceRoutes {
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

func TestConferenceRoutes_InvalidUUIDs_Returns400(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFConferenceScheduling: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})
	tok := confTestToken(t, signer)

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/teachers/not-a-uuid/conference-availability", `{"schoolId":"00000000-0000-0000-0000-000000000001","date":"2025-11-18","windowStart":"16:00","windowEnd":"18:00","slotDuration":15}`},
		{http.MethodGet, "/api/v1/teachers/not-a-uuid/conference-slots?date=2025-11-18", ""},
		{http.MethodPost, "/api/v1/conference-slots/not-a-uuid/book", `{"studentId":"00000000-0000-0000-0000-000000000003"}`},
	}

	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			var req *http.Request
			if c.body != "" {
				req = httptest.NewRequest(c.method, c.path, bytes.NewReader([]byte(c.body)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(c.method, c.path, nil)
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

func TestGenerateSlotTimesMatchesRepo(t *testing.T) {
	// Sanity: repo export matches AC-1 expectations
	if len(conferences.AllowedSlotDurations) != 5 {
		t.Fatalf("expected 5 allowed durations")
	}
}
