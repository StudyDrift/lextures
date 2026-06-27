package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

// Admin job-queue routes require authentication; unauthenticated requests must
// be rejected before any DB access (plan 17.3 §9 admin JWT).
func TestAdminJobs_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}}
	h := NewHandler(d)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/jobs"},
		{http.MethodGet, "/api/v1/admin/jobs/dead-letters"},
		{http.MethodPost, "/api/v1/admin/jobs/dead-letters/00000000-0000-0000-0000-000000000001/redrive"},
		{http.MethodDelete, "/api/v1/admin/jobs/00000000-0000-0000-0000-000000000001"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: want 401 got %d", tc.method, tc.path, w.Code)
		}
	}
}
