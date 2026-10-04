package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestCourseBankQuestionWrites_NotMethodNotAllowed(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.com", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/courses/C-TEST/questions", `{"questionType":"mc_single","stem":"QA route check"}`},
		{http.MethodPut, "/api/v1/courses/C-TEST/questions/00000000-0000-0000-0000-000000000002", `{"stem":"QA route check"}`},
		{http.MethodGet, "/api/v1/courses/C-TEST/questions/00000000-0000-0000-0000-000000000002/versions", ""},
		{http.MethodPost, "/api/v1/courses/C-TEST/questions/00000000-0000-0000-0000-000000000002/versions/1/restore", `{}`},
	}
	for _, tc := range cases {
		var body *strings.Reader
		if tc.body == "" {
			body = strings.NewReader("")
		} else {
			body = strings.NewReader(tc.body)
		}
		req := httptest.NewRequest(tc.method, tc.path, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusMethodNotAllowed || strings.Contains(rr.Body.String(), "method not allowed") {
			t.Fatalf("%s %s method not allowed: %s", tc.method, tc.path, rr.Body.String())
		}
		if rr.Code == http.StatusNotFound && strings.Contains(rr.Body.String(), "No HTTP route") {
			t.Fatalf("%s %s missing route: %s", tc.method, tc.path, rr.Body.String())
		}
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, rr.Code, rr.Body.String())
		}
	}
}

func TestCourseBankQuestionCreate_UnauthenticatedIsNotMethodNotAllowed(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-TEST/questions", strings.NewReader(`{"stem":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "method not allowed") || strings.Contains(rr.Body.String(), "No HTTP route") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}
