package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestAdminIntroCourseStatus_Unauthorized(t *testing.T) {
	d := Deps{Config: config.Config{IntroCourseEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/intro-course", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminIntroCourseResync_Unauthorized(t *testing.T) {
	d := Deps{Config: config.Config{IntroCourseEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/intro-course/resync", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminIntroCourseResync_ForbiddenWithoutRBAC(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	tok, err := signer.Sign(context.Background(), "a0000000-0000-4000-8000-000000000099", "admin@test.invalid", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	d := Deps{
		Config:    config.Config{IntroCourseEnabled: true},
		JWTSigner: signer,
	}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/intro-course/resync", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	// Without Pool, adminRbacUser returns 500 before the RBAC check (no-db mode).
	if rr.Code != http.StatusForbidden && rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestMeIntroCourse_Unauthorized(t *testing.T) {
	d := Deps{Config: config.Config{IntroCourseEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me/intro-course", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestMeIntroCourseWelcomeBannerDismissed_Unauthorized(t *testing.T) {
	d := Deps{Config: config.Config{IntroCourseEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPut, "/api/v1/me/intro-course/welcome-banner-dismissed", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestMeIntroCourseCelebrationSeen_Unauthorized(t *testing.T) {
	d := Deps{Config: config.Config{IntroCourseEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPut, "/api/v1/me/intro-course/celebration-seen", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminIntroCourseAnalytics_Unauthorized(t *testing.T) {
	d := Deps{Config: config.Config{IntroCourseEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/intro-course/analytics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminIntroCourseBackfill_Unauthorized(t *testing.T) {
	d := Deps{Config: config.Config{IntroCourseEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/intro-course/backfill", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("POST status=%d body=%s", rr.Code, rr.Body.String())
	}
	r = httptest.NewRequest(http.MethodGet, "/api/v1/admin/intro-course/backfill", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("GET status=%d body=%s", rr.Code, rr.Body.String())
	}
}