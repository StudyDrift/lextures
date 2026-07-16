package httpserver

import (
	"context"
	"testing"

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
