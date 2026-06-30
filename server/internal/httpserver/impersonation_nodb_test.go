package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestImpersonation_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{AdminConsoleEnabled: true, ImpersonationEnabled: false}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin-console/impersonate", nil)
	rec := httptest.NewRecorder()
	d.handleAdminConsoleImpersonateStart()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestImpersonationWriteBlockMiddleware(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{JWTSigner: signer}
	mw := d.impersonationWriteBlockMiddleware()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mw(next)

	adminID := "11111111-1111-4111-8111-111111111111"
	userID := "22222222-2222-4222-8222-222222222222"
	token, _, err := signer.SignImpersonation(adminID, userID, "s@school.edu", "", "")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/some-write", nil)
	postReq.Header.Set("Authorization", "Bearer "+token)
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusForbidden {
		t.Fatalf("POST status %d body=%s", postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/some-read", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status %d", getRec.Code)
	}

	delExit := httptest.NewRequest(http.MethodDelete, "/api/v1/admin-console/impersonate/session", nil)
	delExit.Header.Set("Authorization", "Bearer "+token)
	delRec := httptest.NewRecorder()
	handler.ServeHTTP(delRec, delExit)
	if delRec.Code != http.StatusOK {
		t.Fatalf("DELETE exit status %d", delRec.Code)
	}
}
