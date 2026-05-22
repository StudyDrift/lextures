package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testCourseCode = "C-ABCDEF"
	testItemID     = "00000000-0000-0000-0000-000000000001"
	baseItemPath   = "/api/v1/courses/" + testCourseCode + "/quizzes/" + testItemID
)

func TestHandleGetItemAnalysis_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, baseItemPath+"/item-analysis", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleGetItemAnalysis_Options(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodOptions, baseItemPath+"/item-analysis", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestHandleComputeItemAnalysis_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodPost, baseItemPath+"/item-analysis/compute", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleComputeItemAnalysis_Options(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodOptions, baseItemPath+"/item-analysis/compute", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestHandleExportItemAnalysisCSV_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, baseItemPath+"/item-analysis/export.csv", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleExportItemAnalysisCSV_Options(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodOptions, baseItemPath+"/item-analysis/export.csv", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestHandleComputeItemAnalysis_WrongMethod(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(http.MethodGet, baseItemPath+"/item-analysis/compute", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	// GET on a POST-only route → 401 (auth check fires before method check in this handler)
	// or 405 depending on routing. Either way it should not be 200.
	if rr.Code == http.StatusOK {
		t.Fatalf("GET on compute endpoint returned 200")
	}
}
