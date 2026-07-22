package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/onboardingevent"
	"github.com/lextures/lextures/server/internal/telemetry"
)

func TestOnboarding_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFOnboardingFlow: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/onboarding-status"},
		{http.MethodPost, "/api/v1/me/onboarding"},
		{http.MethodGet, "/api/v1/me/onboarding/diagnostic-questions?topic=python"},
	}
	for _, tc := range paths {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: want 401 got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestOnboarding_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFOnboardingFlow: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/onboarding-status", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 got %d", w.Code)
	}
}

func TestPublicOnboardingTrack_ProgramValues(t *testing.T) {
	resetOnboardingRateLimiters()
	orig := onboardingInsert
	t.Cleanup(func() {
		onboardingInsert = orig
		resetOnboardingRateLimiters()
	})

	var gotPrograms []string
	onboardingInsert = func(_ context.Context, _ *pgxpool.Pool, e onboardingevent.Event) error {
		gotPrograms = append(gotPrograms, e.Program)
		return nil
	}

	d := Deps{}
	h := d.handlePublicOnboardingTrack()

	accepted := []string{"k-12", "higher-ed", "self-learner", "homeschool", "school"}
	for _, program := range accepted {
		body := `{"program":"` + program + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/public/onboarding/track", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.10:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("program=%q: want 204 got %d body=%s", program, w.Code, w.Body.String())
		}
	}
	if len(gotPrograms) != len(accepted) {
		t.Fatalf("inserts=%d want %d (%v)", len(gotPrograms), len(accepted), gotPrograms)
	}
	for i, want := range accepted {
		if gotPrograms[i] != want {
			t.Fatalf("insert[%d]=%q want %q", i, gotPrograms[i], want)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/onboarding/track", strings.NewReader(`{"program":"bogus"}`))
	req.RemoteAddr = "203.0.113.11:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bogus: want 400 got %d", w.Code)
	}
	if len(gotPrograms) != len(accepted) {
		t.Fatalf("bogus must not insert; got %v", gotPrograms)
	}
}

func TestPublicOnboardingTrack_InsertFailureStill204(t *testing.T) {
	resetOnboardingRateLimiters()
	orig := onboardingInsert
	tel := telemetry.Init(context.Background(), telemetry.Config{})
	t.Cleanup(func() {
		onboardingInsert = orig
		telemetry.SetDefaultForTest(nil)
		resetOnboardingRateLimiters()
	})

	onboardingInsert = func(_ context.Context, _ *pgxpool.Pool, _ onboardingevent.Event) error {
		return errors.New("simulated constraint violation")
	}

	d := Deps{}
	h := d.handlePublicOnboardingTrack()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/onboarding/track", strings.NewReader(`{"program":"homeschool"}`))
	req.RemoteAddr = "203.0.113.12:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204 got %d body=%s", w.Code, w.Body.String())
	}

	rr := httptest.NewRecorder()
	tel.MetricsHandler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rr.Body.String()
	if !strings.Contains(body, `lextures_onboarding_event_insert_failed_total{program="homeschool"} 1`) {
		t.Fatalf("expected insert-failed counter; metrics:\n%s", body)
	}
}

func resetOnboardingRateLimiters() {
	onboardingMu.Lock()
	defer onboardingMu.Unlock()
	onboardingLimiters = map[string]*onboardingIPEntry{}
	onboardingLastClean = time.Now()
}
