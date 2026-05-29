package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestSTTTranscribe_UnauthenticatedReturns401(t *testing.T) {
	d := Deps{Config: config.Config{SpeechToTextEnabled: true}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stt/transcribe", nil)
	rec := httptest.NewRecorder()
	d.handlePostSTTTranscribe()(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestSTTTranscribe_DisabledReturns404(t *testing.T) {
	d := Deps{Config: config.Config{SpeechToTextEnabled: false}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stt/transcribe", nil)
	rec := httptest.NewRecorder()
	d.handlePostSTTTranscribe()(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
