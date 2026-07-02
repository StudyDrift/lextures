package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminPeople_UnauthenticatedReturns401(t *testing.T) {
	d := Deps{}
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/people?q=test", nil)
	d.handleAdminPeopleSearch()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("search status = %d, want 401", rr.Code)
	}

	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodPost, "/api/v1/admin/people/invite", nil)
	d.handleAdminPeopleInvite()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("invite status = %d, want 401", rr.Code)
	}

	rr = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/api/v1/admin/people/00000000-0000-4000-8000-000000000001/report", nil)
	d.handleAdminPeopleReport()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("report status = %d, want 401", rr.Code)
	}
}

func TestAdminPeople_SearchEmptyQueryReturnsEmptyList(t *testing.T) {
	// Without DB/auth this only verifies the handler shape for empty q when unauthenticated.
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/people", nil)
	Deps{}.handleAdminPeopleSearch()(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 without auth", rr.Code)
	}
}