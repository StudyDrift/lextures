package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/platformstate"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

func TestAIConfigured_LegacyOpenRouter(t *testing.T) {
	t.Parallel()
	d := Deps{
		Platform: platformstate.New(config.Config{OpenRouterAPIKey: "sk-test"}),
		Config:   config.Config{OpenRouterAPIKey: "sk-test"},
	}
	if !d.aiConfigured(context.Background(), nil) {
		t.Fatal("expected aiConfigured when OpenRouter key present")
	}
	providers := d.aiProvidersConfigured(context.Background(), nil)
	if len(providers) == 0 {
		t.Fatal("expected at least openrouter in providers list")
	}
	found := false
	for _, p := range providers {
		if p == string(aiprovider.ProviderOpenRouter) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected openrouter in %v", providers)
	}
}

func TestAIConfigured_None(t *testing.T) {
	t.Parallel()
	d := Deps{Config: config.Config{}}
	if d.aiConfigured(context.Background(), nil) {
		t.Fatal("expected aiConfigured false with no credentials")
	}
	if got := d.aiProvidersConfigured(context.Background(), nil); len(got) != 0 {
		t.Fatalf("expected empty providers, got %v", got)
	}
}

func TestWriteAIGenerationFailed_Returns503AndRecordsErr(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/C-TEST/syllabus/generate-section", nil)
	req = apierr.WithServerErrorTracking(req)
	rr := httptest.NewRecorder()
	cause := errors.New("openrouter: status 401: invalid key")

	writeAIGenerationFailed(rr, req, "AI generation failed: "+cause.Error(), cause)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d want %d", rr.Code, http.StatusServiceUnavailable)
	}
	var body apierr.Body
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != apierr.CodeAiGenerationFailed {
		t.Fatalf("code: got %q want %q", body.Error.Code, apierr.CodeAiGenerationFailed)
	}
	if !strings.Contains(body.Error.Message, "invalid key") {
		t.Fatalf("message: got %q", body.Error.Message)
	}
	msg, got := apierr.ServerErrorFromRequest(req)
	if msg != body.Error.Message {
		t.Fatalf("recorded message: got %q want %q", msg, body.Error.Message)
	}
	if got != cause {
		t.Fatalf("recorded err: got %v want %v", got, cause)
	}
}

func TestWriteAIGenerationFailed_TruncatesLongMessage(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	long := strings.Repeat("x", aiGenerationFailedClientMsgMax+50)

	writeAIGenerationFailed(rr, req, long, errors.New("cause"))

	var body apierr.Body
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Error.Message) != aiGenerationFailedClientMsgMax {
		t.Fatalf("message len: got %d want %d", len(body.Error.Message), aiGenerationFailedClientMsgMax)
	}
}
