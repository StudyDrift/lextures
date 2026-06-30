package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIncludeQuery_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	if wantsInclude(req, "custom_fields") {
		t.Fatal("expected false")
	}
}
