package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestHandleCourseFileItemPreview_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Storage: nil})
	itemID := uuid.New()
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST01/files/items/"+itemID.String()+"/preview", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}
