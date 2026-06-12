package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
)

func studentNotebooksTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestStudentNotebooks_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{}
	h := NewHandler(d)
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/me/notebooks"},
		{http.MethodPut, "/api/v1/me/notebooks?courseCode=CS101"},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, strings.NewReader("{}"))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: status=%d want 401", c.method, c.path, w.Code)
		}
	}
}

func TestStudentNotebooks_NoDB_Returns500(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Pool: nil, JWTSigner: signer}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me/notebooks", nil)
	r.Header.Set("Authorization", "Bearer "+studentNotebooksTestToken(t, signer))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want 500", w.Code)
	}
}
