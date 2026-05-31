package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLMSDashboard_learnersStillUnimplementedForOtherPaths(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/learners/abc/xyz", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unrelated learners path: %d", rr.Code)
	}
}

func TestLMSDashboard_courseStructureUnauthorizedWithoutJWT(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST/structure", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("structure: %d", rr.Code)
	}
}

func TestLMSDashboard_syllabusAcceptanceStatusUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST/syllabus/acceptance-status", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("syllabus acceptance: %d", rr.Code)
	}
}

func TestLMSDashboard_getSyllabusUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST/syllabus", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("get syllabus: %d", rr.Code)
	}
}

func TestLMSDashboard_syllabusMarkupsUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST/syllabus/markups", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("syllabus markups: %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestLMSDashboard_feedRosterUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST/feed/roster", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("feed roster: %d", rr.Code)
	}
}

func TestLMSDashboard_feedMessageLikeUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-TEST/feed/messages/00000000-0000-0000-0000-000000000001/like", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("feed like: %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestLMSDashboard_feedMessageUnlikeUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/C-TEST/feed/messages/00000000-0000-0000-0000-000000000001/like", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("feed unlike: %d", rr.Code)
	}
}

func TestLMSDashboard_feedMessagePatchUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/C-TEST/feed/messages/00000000-0000-0000-0000-000000000001", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("feed patch: %d", rr.Code)
	}
}

func TestLMSDashboard_feedMessagePinUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/C-TEST/feed/messages/00000000-0000-0000-0000-000000000001/pin", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("feed pin: %d", rr.Code)
	}
}

func TestLMSDashboard_feedUploadImageUnauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-TEST/feed/upload-image", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("feed upload-image: %d", rr.Code)
	}
}
