package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestCourseEvaluations_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCourseEvaluations: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-FAKE/evaluations/status"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/evaluations/00000000-0000-0000-0000-000000000001/submit"},
		{http.MethodGet, "/api/v1/courses/C-FAKE/evaluations/results"},
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

func TestCourseEvaluations_FeatureOff(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCourseEvaluations: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-FAKE/evaluations/status"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/evaluations/00000000-0000-0000-0000-000000000001/submit"},
		{http.MethodGet, "/api/v1/courses/C-FAKE/evaluations/results"},
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

func TestAdminEvaluationTemplates_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFCourseEvaluations: true}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/evaluation-templates"},
		{http.MethodPost, "/api/v1/admin/evaluation-templates"},
		{http.MethodGet, "/api/v1/admin/evaluation-templates/00000000-0000-0000-0000-000000000001"},
		{http.MethodPatch, "/api/v1/admin/evaluation-templates/00000000-0000-0000-0000-000000000001"},
		{http.MethodDelete, "/api/v1/admin/evaluation-templates/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/admin/courses/C-FAKE/evaluation-windows"},
		{http.MethodGet, "/api/v1/admin/evaluations/report"},
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
