package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/apierr"
)

func TestCourseOutcomes_UnauthenticatedEnvelopeUnchanged(t *testing.T) {
	h := NewHandler(Deps{})
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/courses/X/outcomes"},
		{http.MethodPost, "/api/v1/courses/X/outcomes"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{"title":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status=%d body=%s", tc.method, tc.path, rr.Code, rr.Body.String())
		}
		var env apierr.Body
		if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Error.Code != apierr.CodeUnauthorized || env.Error.Message != "Sign in required." {
			t.Fatalf("%s %s: envelope=%+v", tc.method, tc.path, env)
		}
	}
}
