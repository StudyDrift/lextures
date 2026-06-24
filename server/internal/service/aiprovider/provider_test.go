package aiprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/service/openrouter"
)

func TestResolveModelID(t *testing.T) {
	id, err := ResolveModelID("claude-3-5-sonnet", ProviderAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if id != "claude-3-5-sonnet-20241022" {
		t.Fatalf("anthropic id: %q", id)
	}
}

func TestAnthropicProvider_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"content":[{"text":"Hello from Claude"}],"usage":{"input_tokens":3,"output_tokens":5}}`))
	}))
	defer srv.Close()
	p := NewAnthropicProviderWithBaseURL("key", srv.URL)
	got, err := p.Complete(context.Background(), "claude-3-5-sonnet-20241022", []Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Hello from Claude" {
		t.Fatalf("text: %q", got.Text)
	}
	if got.Usage.PromptTokens != 3 || got.Usage.CompletionTokens != 5 {
		t.Fatalf("usage: %+v", got.Usage)
	}
}

func TestOpenAIProvider_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Hello GPT"}}],"usage":{"prompt_tokens":2,"completion_tokens":4,"total_tokens":6}}`))
	}))
	defer srv.Close()
	p := NewOpenAIProviderWithBaseURL("key", srv.URL)
	got, err := p.Complete(context.Background(), "gpt-4o", []Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Hello GPT" {
		t.Fatalf("text: %q", got.Text)
	}
}

func TestOpenRouterProvider_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"via OR"}}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3,"cost":0.01}}`))
	}))
	defer srv.Close()
	or := openrouter.NewClientWithBaseURL("key", srv.URL+"/v1")
	p := NewOpenRouterProvider(or)
	got, err := p.Complete(context.Background(), "anthropic/claude-3.5-sonnet", []Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "via OR" {
		t.Fatalf("text: %q", got.Text)
	}
}

func TestDryRunProvider_Complete(t *testing.T) {
	p := &DryRunProvider{}
	got, err := p.Complete(context.Background(), "x", []Message{{Role: "user", Content: "test"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text == "" {
		t.Fatal("expected dry-run text")
	}
}

func TestIsRetryable(t *testing.T) {
	if !IsRetryable(newProviderError(ProviderAnthropic, 503, "down")) {
		t.Fatal("503 should be retryable")
	}
	if IsRetryable(newProviderError(ProviderAnthropic, 400, "bad")) {
		t.Fatal("400 should not be retryable")
	}
}

func TestFactory_BuildProviders(t *testing.T) {
	f := Factory{}
	if _, err := f.Build(ProviderAnthropic, "k", nil); err != nil {
		t.Fatalf("anthropic: %v", err)
	}
	if _, err := f.Build(ProviderBedrock, "", map[string]any{"aws_region": "us-west-2"}); err != nil {
		t.Fatalf("bedrock: %v", err)
	}
}