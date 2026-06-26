package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
)

func studentTodosTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestStudentTodoBoard_Unauthenticated_Returns401(t *testing.T) {
	d := Deps{}
	h := NewHandler(d)
	for _, path := range []string{
		"/api/v1/me/student-todo-board",
	} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status=%d want 401", path, w.Code)
		}
	}
}

func TestStudentTodoBoard_NoDB_Returns500(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{Pool: nil, JWTSigner: signer}
	h := NewHandler(d)
	r := httptest.NewRequest(http.MethodGet, "/api/v1/me/student-todo-board", nil)
	r.Header.Set("Authorization", "Bearer "+studentTodosTestToken(t, signer))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want 500", w.Code)
	}
}