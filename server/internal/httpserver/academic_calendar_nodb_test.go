package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

// Auth runs first (requires Pool), so these no-db tests verify the unauthenticated path.
// The authenticated + feature-disabled (501) cases are covered by e2e tests.

func TestCalendarEventsGet_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+uuid.New().String()+"/calendar/events", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminCalendarEventPost_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	body := []byte(`{"eventType":"holiday","eventName":"Test","startDate":"2027-01-01"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orgs/"+uuid.New().String()+"/calendar/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminCalendarEventPatch_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	body := []byte(`{"eventName":"Updated"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orgs/"+uuid.New().String()+"/calendar/events/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAdminCalendarEventDelete_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/orgs/"+uuid.New().String()+"/calendar/events/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCalendarTermICAL_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+uuid.New().String()+"/calendar/terms/"+uuid.New().String()+"/ical", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCalendarRoutes_Registered(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: true}
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: cfg})

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/orgs/" + uuid.New().String() + "/calendar/events"},
		{http.MethodPost, "/api/v1/admin/orgs/" + uuid.New().String() + "/calendar/events"},
		{http.MethodGet, "/api/v1/orgs/" + uuid.New().String() + "/calendar/terms/" + uuid.New().String() + "/ical"},
	}
	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound || rr.Code == http.StatusMethodNotAllowed {
			t.Errorf("%s %s: route not registered (got %d)", rt.method, rt.path, rr.Code)
		}
	}
}

func TestValidCalendarEventType(t *testing.T) {
	valid := []string{"term_start", "term_end", "add_drop_deadline", "withdrawal_deadline", "finals_start", "finals_end", "no_class_day", "holiday", "custom"}
	for _, v := range valid {
		if !validCalendarEventType(v) {
			t.Errorf("expected %q to be valid", v)
		}
	}
	invalid := []string{"", "foo", "Term_Start", "HOLIDAY"}
	for _, v := range invalid {
		if validCalendarEventType(v) {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}

func TestIcalEscape(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello", "hello"},
		{"a,b", `a\,b`},
		{"a;b", `a\;b`},
		{"a\nb", `a\nb`},
		{`a\b`, `a\\b`},
	}
	for _, tc := range cases {
		got := icalEscape(tc.in)
		if got != tc.want {
			t.Errorf("icalEscape(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
