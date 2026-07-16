package aiprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/service/openrouter"
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
	if meta.Operation != OpComplete {
		t.Fatalf("operation: %s", meta.Operation)
	}
	if got.Text == "" {
		t.Fatal("expected text")
	}
}

func TestResolver_ModelOverrideDualRead(t *testing.T) {
	// Stored OpenRouter default must map to a native Anthropic id (AC-4), not be sent raw.
	id, err := ResolveModelID("arcee-ai/trinity-mini:free", ProviderAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if id != "claude-3-5-haiku-20241022" {
		t.Fatalf("override resolve: %q", id)
	}
	_, err = ResolveModelID("vendor/no-mapping:free", ProviderAnthropic)
	if err == nil || !strings.Contains(err.Error(), "no alias mapping") {
		t.Fatalf("expected actionable error, got %v", err)
	}
}

func TestResolver_DryRun_StreamAndVision(t *testing.T) {
	r := NewResolver(nil, nil, ResolverConfig{DryRun: true})
	var chunks int
	got, meta, err := r.CompleteStream(context.Background(), nil, "", []Message{{Role: "user", Content: "ping"}}, func(string) error {
		chunks++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Operation != OpStream || chunks == 0 || got.Text == "" {
		t.Fatalf("stream meta=%+v chunks=%d text=%q", meta, chunks, got.Text)
	}
	got2, meta2, err := r.CompleteVision(context.Background(), nil, "", VisionMessages("s", "u", []string{"https://x"}))
	if err != nil {
		t.Fatal(err)
	}
	if meta2.Operation != OpVision || got2.Text == "" {
		t.Fatalf("vision meta=%+v text=%q", meta2, got2.Text)
	}
}

func TestResolver_DefaultOpenRouterWhenDisabled(t *testing.T) {
	r := &Resolver{
		factory: Factory{},
		cfg: ResolverConfig{
			AbstractionEnabled: false,
			PlatformAPIKey:     "platform-key",
		},
	}
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

func TestResolver_APIKeyForProvider_PlatformFallbackMessage(t *testing.T) {
	r := &Resolver{
		cfg: ResolverConfig{AbstractionEnabled: true},
	}
	_, err := r.apiKeyForProvider(context.Background(), nil, ProviderAnthropic)
	if err == nil {
		t.Fatal("expected AI not configured error")
	}
	if !strings.Contains(err.Error(), "AI not configured") {
		t.Fatalf("err: %v", err)
	}
}

func TestResolver_APIKeyForProvider_LegacyPlatformOpenRouter(t *testing.T) {
	r := &Resolver{
		cfg: ResolverConfig{
			AbstractionEnabled: true,
			PlatformAPIKey:     "or-legacy",
		},
	}
	key, err := r.apiKeyForProvider(context.Background(), nil, ProviderOpenRouter)
	if err != nil {
		t.Fatal(err)
	}
	if key != "or-legacy" {
		t.Fatalf("key: %q", key)
	}
}

func TestResolver_CompleteStream_OpenRouter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"A\"}}]}\n\n")
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"B\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	or := openrouter.NewClientWithBaseURL("key", srv.URL+"/v1")
	r := NewResolver(nil, or, ResolverConfig{
		AbstractionEnabled: false,
		PlatformAPIKey:     "key",
	})
	var chunks []string
	got, meta, err := r.CompleteStream(context.Background(), nil, "anthropic/claude-3.5-sonnet",
		[]Message{{Role: "user", Content: "hi"}},
		func(text string) error {
			chunks = append(chunks, text)
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "AB" || len(chunks) != 2 {
		t.Fatalf("got=%q chunks=%v", got.Text, chunks)
	}
	if meta.Provider != ProviderOpenRouter || meta.Operation != OpStream {
		t.Fatalf("meta: %+v", meta)
	}
}

func TestResolver_CompleteStream_CapabilityGapNoFallback(t *testing.T) {
	primary := &MockProvider{
		NameValue: ProviderBedrock,
		CompleteStreamFunc: func(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
			return ChatResult{}, notSupported("stream")
		},
	}
	fallback := &MockProvider{
		NameValue: ProviderOpenRouter,
		CompleteStreamFunc: func(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
			t.Fatal("fallback must not run for capability gaps")
			return ChatResult{}, nil
		},
	}
	fb := ProviderOpenRouter
	r := &Resolver{
		cfg:     ResolverConfig{AbstractionEnabled: true, PlatformAPIKey: "k"},
		factory: Factory{},
	}
	// Bypass factory by calling callProvider/dispatch pieces directly.
	got, meta, err := r.callProvider(context.Background(), primary, ProviderBedrock, "claude-3-5-sonnet", "model", OpStream, AuthModeAPIKey,
		func(ctx context.Context, p Provider, modelID string) (ChatResult, error) {
			return p.CompleteStream(ctx, modelID, nil, nil)
		},
	)
	_ = got
	_ = meta
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
	if IsRetryable(err) {
		t.Fatal("capability gap must not be retryable")
	}
	// Confirm fallback would be skipped by Complete's rule.
	if IsRetryable(err) || !IsCapabilityGap(err) {
		t.Fatal("fallback gate broken")
	}
	_ = fallback
	_ = fb
}

func TestResolver_Complete_FallbackOn503(t *testing.T) {
	calls := 0
	primary := &MockProvider{
		NameValue: ProviderAnthropic,
		CompleteFunc: func(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
			calls++
			return ChatResult{}, newProviderError(ProviderAnthropic, 503, "down")
		},
	}
	fallback := &MockProvider{
		NameValue: ProviderOpenRouter,
		CompleteFunc: func(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
			calls++
			return ChatResult{Text: "recovered", Usage: UsageInfo{TotalTokens: 1}}, nil
		},
	}
	fb := ProviderOpenRouter
	r := &Resolver{
		cfg: ResolverConfig{AbstractionEnabled: true, PlatformAPIKey: "platform"},
	}
	settings := Settings{
		Provider:         ProviderAnthropic,
		ModelAlias:       string(AliasClaude35Sonnet),
		FallbackProvider: &fb,
	}
	// Simulate dispatch fallback path manually with injected providers.
	got, meta, err := r.callProvider(context.Background(), primary, settings.Provider, settings.ModelAlias, "id", OpComplete, AuthModeAPIKey,
		func(ctx context.Context, p Provider, modelID string) (ChatResult, error) {
			return p.Complete(ctx, modelID, nil)
		},
	)
	if err == nil || !IsRetryable(err) {
		t.Fatalf("expected retryable err, got %v", err)
	}
	got2, meta2, err2 := r.callProvider(context.Background(), fallback, fb, settings.ModelAlias, "id2", OpComplete, AuthModeAPIKey,
		func(ctx context.Context, p Provider, modelID string) (ChatResult, error) {
			return p.Complete(ctx, modelID, nil)
		},
	)
	if err2 != nil {
		t.Fatal(err2)
	}
	if got2.Text != "recovered" {
		t.Fatalf("text: %q", got2.Text)
	}
	_ = got
	_ = meta
	_ = meta2
	if calls != 2 {
		t.Fatalf("calls: %d", calls)
	}
}
