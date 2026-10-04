package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestGradebookCSVRoutes_Registered(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{GradebookCSVEnabled: true}})
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-HUPCNF/gradebook.csv"},
		{http.MethodPost, "/api/v1/courses/C-HUPCNF/gradebook/import/validate"},
		{http.MethodPost, "/api/v1/courses/C-HUPCNF/gradebook/import/confirm"},
		{http.MethodDelete, "/api/v1/courses/C-HUPCNF/gradebook/import/00000000-0000-0000-0000-000000000002"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code == http.StatusNotFound {
				t.Fatalf("route not registered, got 404: %s", rr.Body.String())
			}
		})
	}
}

func TestGradebookCSVRoutes_Unauthenticated(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{GradebookCSVEnabled: true}})
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/courses/C-HUPCNF/gradebook.csv"},
		{http.MethodPost, "/api/v1/courses/C-HUPCNF/gradebook/import/validate"},
		{http.MethodPost, "/api/v1/courses/C-HUPCNF/gradebook/import/confirm"},
		{http.MethodDelete, "/api/v1/courses/C-HUPCNF/gradebook/import/00000000-0000-0000-0000-000000000002"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: got %d, want 401 (%s)", c.method, c.path, rr.Code, rr.Body.String())
		}
	}
}
