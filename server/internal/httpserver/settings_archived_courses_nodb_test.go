package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSettingsArchivedCourses_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	for _, p := range []struct {
		path   string
		method string
	}{
		{"/api/v1/settings/archived-courses", http.MethodGet},
		{"/api/v1/settings/archived-courses/C-TEST01/restore", http.MethodPost},
		{"/api/v1/settings/archived-courses/C-TEST01", http.MethodDelete},
	} {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(p.method, p.path, nil)
		h.ServeHTTP(rr, r)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status=%d want 401", p.method, p.path, rr.Code)
		}
	}
}