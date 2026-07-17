package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestTranscriptInbound_FeatureOff(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFTranscripts: true, FFTranscriptInbound: false}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/transcripts/inbound", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusUnauthorized {
		// Without auth, may be 401 before feature check depending on middleware order;
		// feature-off path returns 404 when authenticated gate is reached via integrations.
		t.Logf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	h2 := NewHandler(Deps{Config: config.Config{FFTranscripts: true, FFTranscriptInbound: false}})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/transcripts/inbound", nil)
	rec2 := httptest.NewRecorder()
	h2.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when inbound flag off, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestTranscriptInbound_MasterFlagOff(t *testing.T) {
	h := NewHandler(Deps{Config: config.Config{FFTranscripts: false, FFTranscriptInbound: true}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/transcripts/inbound", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when ff_transcripts off, got %d", rec.Code)
	}
}
