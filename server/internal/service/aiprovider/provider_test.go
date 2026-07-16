package aiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCapabilitiesMatrix(t *testing.T) {
	or := Capabilities(ProviderOpenRouter)
	if !or.Stream || !or.Vision || !or.Image || or.Embed {
		t.Fatalf("openrouter caps: %+v", or)
	}
	bedrock := Capabilities(ProviderBedrock)
	if bedrock.Stream || bedrock.Vision {
		t.Fatalf("bedrock should lack stream/vision: %+v", bedrock)
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

func TestAnthropicProvider_Complete_JSONMode(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &gotBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		_, _ = w.Write([]byte(`{"content":[{"text":"{\"ok\":true}"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer srv.Close()
	p := NewAnthropicProviderWithBaseURL("key", srv.URL)
	temp := 0.2
	_, err := p.Complete(context.Background(), "claude-3-5-sonnet-20241022",
		[]Message{{Role: "system", Content: "Grade this"}, {Role: "user", Content: "essay"}},
		ChatOptions{JSONMode: true, MaxTokens: 256, Temperature: &temp},
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["max_tokens"].(float64) != 256 {
		t.Fatalf("max_tokens: %v", gotBody["max_tokens"])
	}
	if gotBody["temperature"].(float64) != 0.2 {
		t.Fatalf("temperature: %v", gotBody["temperature"])
	}
	sys, _ := gotBody["system"].(string)
	if !strings.Contains(sys, "Respond with valid JSON only.") {
		t.Fatalf("system missing JSON instruction: %q", sys)
	}
	if _, ok := gotBody["response_format"]; ok {
		t.Fatal("anthropic must not send OpenAI response_format")
	}
	msgs, _ := gotBody["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected system extracted from messages, got %d msgs", len(msgs))
	}
}

func TestAnthropicProvider_CompleteStream_NotSupported(t *testing.T) {
	p := NewAnthropicProviderWithBaseURL("key", "http://127.0.0.1:1")
	_, err := p.CompleteStream(context.Background(), "m", nil, nil)
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
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

func TestOpenAIProvider_CompleteStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	p := NewOpenAIProviderWithBaseURL("key", srv.URL)
	var chunks []string
	got, err := p.CompleteStream(context.Background(), "gpt-4o", []Message{{Role: "user", Content: "x"}}, func(text string) error {
		chunks = append(chunks, text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Hi" || len(chunks) != 1 {
		t.Fatalf("got=%q chunks=%v", got.Text, chunks)
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

func TestOpenRouterProvider_CompleteStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{"Hello", " ", "world"}
		for _, ch := range chunks {
			_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", ch)
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	or := openrouter.NewClientWithBaseURL("key", srv.URL+"/v1")
	p := NewOpenRouterProvider(or)
	var got []string
	full, err := p.CompleteStream(context.Background(), "m", []Message{{Role: "user", Content: "hi"}}, func(text string) error {
		got = append(got, text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if full.Text != "Hello world" {
		t.Fatalf("full: %q", full.Text)
	}
	if len(got) != 3 {
		t.Fatalf("chunks: %v", got)
	}
}

func TestAnthropicProvider_CompleteVision(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"content":[{"text":"vision ok"}],"usage":{"input_tokens":4,"output_tokens":2}}`))
	}))
	defer srv.Close()
	p := NewAnthropicProviderWithBaseURL("key", srv.URL)
	msgs := VisionMessages("Describe", "what is this", []string{"data:image/png;base64,aaa"})
	got, err := p.CompleteVision(context.Background(), "claude-3-5-sonnet-20241022", msgs)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "vision ok" {
		t.Fatalf("text: %q", got.Text)
	}
	msgsRaw, _ := gotBody["messages"].([]any)
	if len(msgsRaw) == 0 {
		t.Fatalf("expected vision messages in body: %v", gotBody)
	}
}

func TestOpenAIProvider_CompleteVision(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"openai vision"}}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`))
	}))
	defer srv.Close()
	p := NewOpenAIProviderWithBaseURL("key", srv.URL)
	msgs := VisionMessages("sys", "look", []string{"https://cdn.example/img.png"})
	got, err := p.CompleteVision(context.Background(), "gpt-4o", msgs)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "openai vision" {
		t.Fatalf("text: %q", got.Text)
	}
	if gotBody["model"] != "gpt-4o" {
		t.Fatalf("model: %v", gotBody["model"])
	}
}

func TestOpenRouterProvider_CompleteVision(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"alt text"}}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`))
	}))
	defer srv.Close()
	or := openrouter.NewClientWithBaseURL("key", srv.URL+"/v1")
	p := NewOpenRouterProvider(or)
	msgs := VisionMessages("Describe", "what is this", []string{"data:image/png;base64,aaa", "https://cdn.example/img.png"})
	got, err := p.CompleteVision(context.Background(), "openai/gpt-4o", msgs, ChatOptions{JSONMode: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "alt text" {
		t.Fatalf("text: %q", got.Text)
	}
	msgsRaw, _ := gotBody["messages"].([]any)
	if len(msgsRaw) != 2 {
		t.Fatalf("messages: %v", msgsRaw)
	}
	rf, _ := gotBody["response_format"].(map[string]any)
	if rf["type"] != "json_object" {
		t.Fatalf("json mode not set: %v", gotBody["response_format"])
	}
}

func TestOpenRouterProvider_GenerateImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"url":"https://cdn.example/out.png"}]}`))
	}))
	defer srv.Close()
	or := openrouter.NewClientWithBaseURL("key", srv.URL+"/v1")
	p := NewOpenRouterProvider(or)
	got, err := p.GenerateImage(context.Background(), "black-forest-labs/flux.2-flex", "a cat")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.URLs) != 1 || got.URLs[0] != "https://cdn.example/out.png" {
		t.Fatalf("urls: %v", got.URLs)
	}
}

func TestDryRunProvider_AllMethods(t *testing.T) {
	p := &DryRunProvider{}
	ctx := context.Background()
	msgs := VisionMessages("sys", "look", []string{"https://x/y.png"})

	if _, err := p.Complete(ctx, "x", msgs); err != nil {
		t.Fatal(err)
	}
	var chunks int
	if _, err := p.CompleteStream(ctx, "x", msgs, func(string) error { chunks++; return nil }); err != nil {
		t.Fatal(err)
	}
	if chunks == 0 {
		t.Fatal("expected stream chunks")
	}
	if _, err := p.CompleteVision(ctx, "x", msgs); err != nil {
		t.Fatal(err)
	}
	if emb, err := p.Embed(ctx, "t"); err != nil || len(emb) == 0 {
		t.Fatalf("embed: %v %v", emb, err)
	}
	if img, err := p.GenerateImage(ctx, "m", "p"); err != nil || len(img.URLs) == 0 {
		t.Fatalf("image: %v %v", img, err)
	}
}

func TestIsRetryable(t *testing.T) {
	if !IsRetryable(newProviderError(ProviderAnthropic, 503, "down")) {
		t.Fatal("503 should be retryable")
	}
	if IsRetryable(newProviderError(ProviderAnthropic, 400, "bad")) {
		t.Fatal("400 should not be retryable")
	}
	if IsRetryable(notSupported("stream")) {
		t.Fatal("capability gap must not be retryable")
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

func TestBedrockProvider_CompleteStream_NotSupported(t *testing.T) {
	p := NewBedrockProvider("", "http://example")
	_, err := p.CompleteStream(context.Background(), "m", nil, nil)
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("got %v", err)
	}
}

func TestProviderInterfaceCompile(t *testing.T) {
	var _ Provider = (*OpenRouterProvider)(nil)
	var _ Provider = (*AnthropicProvider)(nil)
	var _ Provider = (*OpenAIProvider)(nil)
	var _ Provider = (*BedrockProvider)(nil)
	var _ Provider = (*VertexProvider)(nil)
	var _ Provider = (*DryRunProvider)(nil)
	var _ Provider = (*MockProvider)(nil)
	var _ ImageProvider = (*OpenRouterProvider)(nil)
	var _ ImageProvider = (*DryRunProvider)(nil)
	var _ ImageProvider = (*MockProvider)(nil)
}
