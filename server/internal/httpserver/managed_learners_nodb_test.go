package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestManagedLearners_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{FFHomeschoolManagedLearners: false}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/dependents", nil)
	rec := httptest.NewRecorder()
	d.handleListDependents()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}

	coursesReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/dependents/courses", nil)
	coursesRec := httptest.NewRecorder()
	d.handleListDependentCourses()(coursesRec, coursesReq)
	if coursesRec.Code != http.StatusNotFound {
		t.Fatalf("courses status %d", coursesRec.Code)
	}
}

func TestManagedLearnerGuardMiddleware(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{JWTSigner: signer, Config: config.Config{FFHomeschoolManagedLearners: true}}
	mw := d.managedLearnerGuardMiddleware()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mw(next)

	actor := "11111111-1111-4111-8111-111111111111"
	child := "22222222-2222-4222-8222-222222222222"
	token, _, err := signer.SignManagedLearner(actor, child, child+"@managed.lextures.invalid", "", "")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	postDeps := httptest.NewRequest(http.MethodPost, "/api/v1/me/dependents", nil)
	postDeps.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, postDeps)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST dependents status %d", rec.Code)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/users", nil)
	adminReq.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, adminReq)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("admin status %d", rec2.Code)
	}

	courseReq := httptest.NewRequest(http.MethodPost, "/api/v1/courses/abc/quiz-submissions", nil)
	courseReq.Header.Set("Authorization", "Bearer "+token)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, courseReq)
	if rec3.Code != http.StatusOK {
		t.Fatalf("course write should be allowed, got %d", rec3.Code)
	}

	exitReq := httptest.NewRequest(http.MethodDelete, "/api/v1/me/dependents/sessions/current", nil)
	exitReq.Header.Set("Authorization", "Bearer "+token)
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, exitReq)
	if rec4.Code != http.StatusOK {
		t.Fatalf("exit status %d", rec4.Code)
	}
}
