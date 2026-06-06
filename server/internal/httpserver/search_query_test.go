package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleSearchQuery_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/search/query?q=hello", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code: %d", rr.Code)
	}
}

func TestParseSearchQueryTypes_Defaults(t *testing.T) {
	types := parseSearchQueryTypes("")
	if !types["course"] || !types["person"] || !types["content"] {
		t.Fatalf("defaults: %+v", types)
	}
}

func TestCourseCodesForPermissions_Wildcard(t *testing.T) {
	codes, all := courseCodesForPermissions([]string{"course:*:enrollments:read"}, "enrollments:read")
	if !all || len(codes) != 0 {
		t.Fatalf("wildcard: all=%v codes=%+v", all, codes)
	}
}
