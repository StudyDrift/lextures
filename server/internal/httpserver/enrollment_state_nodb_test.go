package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

func enrollmentStateTestToken(t *testing.T, signer *auth.JWTSigner) string {
	t.Helper()
	tok, err := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.invalid", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestEnrollmentStatePatch_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFEnrollmentStateMachine: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := enrollmentStateTestToken(t, signer)

	r := chi.NewRouter()
	r.Patch("/api/v1/courses/{course_code}/enrollments/{enrollment_id}/state", d.handleEnrollmentStatePatch())

	body, _ := json.Marshal(map[string]string{"state": "withdrawn"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/courses/CS101/enrollments/"+uuid.New().String()+"/state", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}

func TestEnrollmentStateHistory_FeatureDisabled(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	cfg := config.Config{FFEnrollmentStateMachine: false}
	d := Deps{Pool: nil, JWTSigner: signer, Config: cfg}
	tok := enrollmentStateTestToken(t, signer)

	r := chi.NewRouter()
	r.Get("/api/v1/courses/{course_code}/enrollments/{enrollment_id}/state/history", d.handleEnrollmentStateHistory())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/CS101/enrollments/"+uuid.New().String()+"/state/history", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", w.Code)
	}
}
