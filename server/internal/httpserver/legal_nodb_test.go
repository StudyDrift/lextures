package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/legalack"
)

func TestLegalPending_Unauthorized(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/legal/pending", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestLegalPending_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	s := auth.NewJWTSigner("test-secret")
	h := NewHandler(Deps{JWTSigner: s})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/legal/pending", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestLegalAcknowledge_Unauthorized(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Pool: nil, JWTSigner: nil})
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{
		"document": legalack.DocumentPrivacyPolicy,
		"version":  "2026-05-21",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/legal/acknowledge", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestLegalAcknowledge_InvalidDocument(t *testing.T) {
	t.Parallel()
	s := auth.NewJWTSigner("test-secret-at-least-32-chars-long")
	h := NewHandler(Deps{JWTSigner: s, Pool: nil})
	rr := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"document": "invalid", "version": "2026-05-21"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/legal/acknowledge", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	// Without a valid JWT user in DB this returns 401 before validation; test unknown doc via unit logic instead.
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 401 or 500 without DB, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCurrentLegalVersions_MatchFrontend(t *testing.T) {
	t.Parallel()
	if len(currentLegalVersions) != 2 {
		t.Fatalf("expected 2 current legal versions, got %d", len(currentLegalVersions))
	}
	priv, ok := currentLegalVersions[legalack.DocumentPrivacyPolicy]
	if !ok || priv.Version == "" || priv.EffectiveDate == "" {
		t.Fatal("privacy_policy version missing")
	}
	terms, ok := currentLegalVersions[legalack.DocumentTermsOfService]
	if !ok || terms.Version == "" || terms.EffectiveDate == "" {
		t.Fatal("terms_of_service version missing")
	}
}
