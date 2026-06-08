package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func academicCalendarTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(t.Context(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestCalendarEventsGet_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := academicCalendarTestToken(t, signer)

	r := chi.NewRouter()
	r.Get("/api/v1/orgs/{orgId}/calendar/events", d.handleCalendarEventsGet())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+uuid.New().String()+"/calendar/events", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestAdminCalendarEventPost_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := academicCalendarTestToken(t, signer)

	r := chi.NewRouter()
	r.Post("/api/v1/admin/orgs/{orgId}/calendar/events", d.handleAdminCalendarEventPost())

	body, _ := json.Marshal(map[string]string{
		"eventType": "holiday",
		"eventName": "Test Holiday",
		"startDate": "2027-01-01",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orgs/"+uuid.New().String()+"/calendar/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestAdminCalendarEventPatch_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := academicCalendarTestToken(t, signer)

	r := chi.NewRouter()
	r.Patch("/api/v1/admin/orgs/{orgId}/calendar/events/{eventId}", d.handleAdminCalendarEventPatch())

	body, _ := json.Marshal(map[string]string{"eventName": "Updated"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orgs/"+uuid.New().String()+"/calendar/events/"+uuid.New().String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestAdminCalendarEventDelete_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := academicCalendarTestToken(t, signer)

	r := chi.NewRouter()
	r.Delete("/api/v1/admin/orgs/{orgId}/calendar/events/{eventId}", d.handleAdminCalendarEventDelete())

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/orgs/"+uuid.New().String()+"/calendar/events/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestCalendarTermICAL_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFAcademicCalendar: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := academicCalendarTestToken(t, signer)

	r := chi.NewRouter()
	r.Get("/api/v1/orgs/{orgId}/calendar/terms/{termId}/ical", d.handleCalendarTermICAL())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+uuid.New().String()+"/calendar/terms/"+uuid.New().String()+"/ical", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
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
