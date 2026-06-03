package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCourseAttendance_Unauthenticated(t *testing.T) {
	d := Deps{}
	h := NewHandler(d)
	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-FAKE/attendance/sessions"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/attendance/sessions"},
		{http.MethodGet, "/api/v1/courses/C-FAKE/attendance/sessions/00000000-0000-0000-0000-000000000001"},
		{http.MethodPut, "/api/v1/courses/C-FAKE/attendance/sessions/00000000-0000-0000-0000-000000000001/records"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/attendance/sessions/00000000-0000-0000-0000-000000000001/self-report"},
		{http.MethodPost, "/api/v1/courses/C-FAKE/attendance/sessions/00000000-0000-0000-0000-000000000001/close"},
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
