package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestCourseReviews_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCourseReviews: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-FAKE/reviews/eligibility"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/reviews"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/reviews/00000000-0000-0000-0000-000000000001/flag"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/reviews/00000000-0000-0000-0000-000000000001/response"},
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

func TestCourseReviews_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCourseReviews: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-FAKE/reviews"},
		{http.MethodGet, "/api/v1/courses/C-FAKE/reviews/eligibility"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/reviews"},
		{http.MethodGet, "/api/v1/admin/reviews"},
		{http.MethodDelete, "/api/v1/admin/reviews/00000000-0000-0000-0000-000000000001"},
	}
	for _, tc := range paths {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s %s: want 404 got %d", tc.method, tc.path, w.Code)
		}
	}
}

func TestCourseReviews_ListPublic_NoAuth(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCourseReviews: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-FAKE/reviews", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	// Pool nil => internal error or not found depending on course lookup; must not be 401.
	if w.Code == http.StatusUnauthorized {
		t.Fatalf("public list must not require auth, got 401")
	}
}

func TestAdminReviews_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCourseReviews: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/reviews"},
		{http.MethodDelete, "/api/v1/admin/reviews/00000000-0000-0000-0000-000000000001"},
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
