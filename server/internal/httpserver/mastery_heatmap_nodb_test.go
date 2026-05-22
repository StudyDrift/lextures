package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
)

func TestMasteryHeatmap_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/analytics/mastery-heatmap", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestMasteryHeatmap_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/CS101/analytics/mastery-heatmap", nil)
	r.Header.Set("Authorization", "Bearer x")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestMasteryHeatmapRefresh_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/courses/CS101/analytics/mastery-heatmap/refresh", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestMasteryHeatmapRefresh_MethodNotAllowed(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/analytics/mastery-heatmap/refresh", nil)
	r.Header.Set("Authorization", "Bearer x")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rr.Code)
	}
}

func TestEnrollmentMastery_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/enrollments/00000000-0000-0000-0000-000000000001/mastery", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rr.Code)
	}
}

func TestMasteryHeatmapRouteRegistered(t *testing.T) {
	s := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: s})

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/CS101/analytics/mastery-heatmap"},
		{http.MethodGet, "/api/v1/courses/CS101/analytics/mastery-heatmap/concepts/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/courses/CS101/enrollments/00000000-0000-0000-0000-000000000001/mastery"},
		{http.MethodPost, "/api/v1/courses/CS101/analytics/mastery-heatmap/refresh"},
	}
	for _, tc := range tests {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		// No bearer — auth runs first, expect 401 not 404.
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound {
			t.Fatalf("route not registered: %s %s", tc.method, tc.path)
		}
	}
}
