package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminCourses_UnauthenticatedReturns401(t *testing.T) {
	d := Deps{}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/courses?q=test", nil)
	d.handleAdminCoursesSearch()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("search status = %d, want 401", rr.Code)
	}

	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/api/v1/admin/courses/00000000-0000-4000-8000-000000000001/report", nil)
	d.handleAdminCoursesReport()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("report status = %d, want 401", rr.Code)
	}

	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/api/v1/admin/courses/00000000-0000-4000-8000-000000000001/access", nil)
	d.handleAdminCoursesAccess()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("access status = %d, want 401", rr.Code)
	}
}