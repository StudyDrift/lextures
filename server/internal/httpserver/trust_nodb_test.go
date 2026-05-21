package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustSubscribe_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/trust/sub-processor-updates/subscribe", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestTrustSubscribe_MissingEmail(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{})
	body, _ := json.Marshal(map[string]string{})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trust/sub-processor-updates/subscribe", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing email, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTrustSubscribe_InvalidJSON(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trust/sub-processor-updates/subscribe", bytes.NewBufferString("{bad json"))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestTrustSubscribe_NoDB_Returns503(t *testing.T) {
	t.Parallel()
	h := NewHandler(Deps{Pool: nil})
	body, _ := json.Marshal(map[string]string{"email": "test@example.com"})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/trust/sub-processor-updates/subscribe", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without DB pool, got %d body=%s", rr.Code, rr.Body.String())
	}
}
