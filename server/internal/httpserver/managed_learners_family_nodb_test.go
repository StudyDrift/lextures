package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func TestFamilySignIn_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{FFHomeschoolManagedLearners: false}}
	for _, h := range []http.HandlerFunc{d.handleFamilyPicker(), d.handleFamilySignIn(), d.handleRegisterFamilyDevice()} {
		req := httptest.NewRequest(http.MethodPost, familySignInPickerPath, strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d", rec.Code)
		}
	}
}

func TestFamilySignIn_InvalidLearnerID(t *testing.T) {
	d := Deps{Config: config.Config{FFHomeschoolManagedLearners: true}}
	req := httptest.NewRequest(http.MethodPost, familySignInSessionPath, strings.NewReader(`{"deviceToken":"x","learnerId":"nope","pin":"1234"}`))
	rec := httptest.NewRecorder()
	d.handleFamilySignIn()(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestFamilySignIn_SessionPathIsAuthRateLimited(t *testing.T) {
	if !authSensitivePaths[familySignInSessionPath] {
		t.Fatal("learner PIN sign-in must use the strict auth rate limit")
	}
}

// Learner sessions started from a family device keep every Learn-as restriction (#771).
func TestManagedLearnerGuard_FamilyDeviceSession(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	d := Deps{JWTSigner: signer, Config: config.Config{FFHomeschoolManagedLearners: true}}
	handler := d.managedLearnerGuardMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	actor := "11111111-1111-4111-8111-111111111111"
	child := "22222222-2222-4222-8222-222222222222"
	token, _, err := signer.SignManagedLearnerVia(actor, child, child+"@managed.lextures.invalid", "", "", auth.ManagedLearnerViaFamilyDevice)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	cases := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/v1/me/dependents/family-devices", http.StatusForbidden},
		{http.MethodPost, "/api/v1/me/dependents/family-devices", http.StatusForbidden},
		{http.MethodPut, "/api/v1/me/dependents/" + child + "/sign-in", http.StatusForbidden},
		{http.MethodDelete, "/api/v1/me/dependents/" + child + "/sign-in", http.StatusForbidden},
		{http.MethodPost, "/api/v1/billing/checkout", http.StatusForbidden},
		{http.MethodGet, "/api/v1/admin-console/users", http.StatusForbidden},
		{http.MethodDelete, "/api/v1/me/dependents/sessions/current", http.StatusOK},
		{http.MethodGet, "/api/v1/me", http.StatusOK},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Fatalf("%s %s: status %d want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}
