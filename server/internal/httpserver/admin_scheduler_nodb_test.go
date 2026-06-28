package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

// Admin scheduler routes require authentication; unauthenticated requests must
// be rejected before any DB or scheduler access (plan 17.4 §9 admin JWT, NFR
// security: no unauthenticated trigger endpoint).
func TestAdminScheduler_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}}
	h := NewHandler(d)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/scheduler"},
		{http.MethodGet, "/api/v1/admin/scheduler/late_submission_sweep/history"},
		{http.MethodPost, "/api/v1/admin/scheduler/late_submission_sweep/enable"},
		{http.MethodPost, "/api/v1/admin/scheduler/late_submission_sweep/disable"},
		{http.MethodPost, "/api/v1/admin/scheduler/late_submission_sweep/trigger"},
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
