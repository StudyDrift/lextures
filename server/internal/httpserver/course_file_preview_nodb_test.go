package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestHandleCourseFilePreview_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Storage: nil})
	fileID := uuid.New()
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST01/course-files/"+fileID.String()+"/preview", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}