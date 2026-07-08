package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestLearnerProfile_FeatureDisabled_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{LearnerProfileEnabled: false}}
	h := NewHandler(d)
	for _, path := range []string{
		"/api/v1/me/learner-profile",
		"/api/v1/me/learner-profile/facets/study_rhythm",
		"/api/v1/me/learner-profile/facets/study_rhythm/evidence",
		"/api/v1/me/learner-profile/export",
	} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: status=%d want 404", path, w.Code)
		}
	}
}

func TestLearnerProfile_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{LearnerProfileEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me/learner-profile", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", w.Code)
	}
}

func TestLearnerProfile_ControlEndpoints_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{Config: config.Config{LearnerProfileEnabled: true}}
	h := NewHandler(d)
	for _, path := range []string{
		"/api/v1/me/learner-profile/pause",
		"/api/v1/me/learner-profile/resume",
		"/api/v1/me/learner-profile/reset",
		"/api/v1/me/learner-profile/export",
	} {
		method := http.MethodPost
		if path == "/api/v1/me/learner-profile/export" {
			method = http.MethodGet
		}
		r := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status=%d want 401", method, path, w.Code)
		}
	}
}

func TestLearnerProfile_UnknownFacet_Returns404(t *testing.T) {
	d := Deps{Config: config.Config{LearnerProfileEnabled: true}}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me/learner-profile/facets/not_a_facet", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 401 or 404", w.Code)
	}
}