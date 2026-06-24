package aiprovider

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestResolver_DryRun(t *testing.T) {
	r := NewResolver(nil, nil, ResolverConfig{DryRun: true})
	got, meta, err := r.Complete(context.Background(), nil, "", []Message{{Role: "user", Content: "ping"}})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Provider != ProviderDryRun {
		t.Fatalf("provider: %s", meta.Provider)
	}
	if got.Text == "" {
		t.Fatal("expected text")
	}
}

func TestResolver_DefaultOpenRouterWhenDisabled(t *testing.T) {
	mock := &MockProvider{
		NameValue: ProviderOpenRouter,
		CompleteFunc: func(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
			return ChatResult{Text: "ok", Usage: UsageInfo{TotalTokens: 1}}, nil
		},
	}
	// Resolver with abstraction disabled should not need DB.
	r := &Resolver{
		factory: Factory{},
		cfg: ResolverConfig{
			AbstractionEnabled: false,
			PlatformAPIKey:     "platform-key",
		},
	}
	// Inject mock via factory override is not exposed; use dry-run path validated above.
	_ = mock
	org := uuid.New()
	settings, key, err := r.resolveTenant(context.Background(), &org)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Provider != ProviderOpenRouter {
		t.Fatalf("provider: %s", settings.Provider)
	}
	if key != "platform-key" {
		t.Fatalf("key: %q", key)
	}
}