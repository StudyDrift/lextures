package captioning_test

import (
	"context"
	"testing"

	"github.com/lextures/lextures/server/internal/workers/captioning"
)

func TestNew_Defaults(t *testing.T) {
	w := captioning.New(nil, nil, captioning.BackendStub, "")
	if w == nil {
		t.Fatal("New returned nil")
	}
	if w.MaxAttempts <= 0 {
		t.Errorf("MaxAttempts should be positive, got %d", w.MaxAttempts)
	}
	if w.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestProcessNext_NoPool(t *testing.T) {
	w := captioning.New(nil, nil, captioning.BackendStub, "")
	_, err := w.ProcessNext(context.Background())
	if err == nil {
		t.Error("expected error when pool is nil, got nil")
	}
}

func TestProcessNext_NoStorage(t *testing.T) {
	w := &captioning.Worker{
		Pool:        nil,
		Storage:     nil,
		Backend:     captioning.BackendStub,
		MaxAttempts: 3,
	}
	_, err := w.ProcessNext(context.Background())
	if err == nil {
		t.Error("expected error when pool is nil, got nil")
	}
}

func TestBackendConstants(t *testing.T) {
	backends := []captioning.Backend{
		captioning.BackendWhisperAPI,
		captioning.BackendWhisperLocal,
		captioning.BackendAzureSpeech,
		captioning.BackendGoogleSpeech,
		captioning.BackendStub,
	}
	seen := map[captioning.Backend]bool{}
	for _, b := range backends {
		if seen[b] {
			t.Errorf("duplicate backend constant: %q", b)
		}
		if string(b) == "" {
			t.Error("backend constant must not be empty string")
		}
		seen[b] = true
	}
}
