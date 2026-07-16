package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/openrouter"
)

// OpenRouterProvider adapts the existing OpenRouter client.
type OpenRouterProvider struct {
	client *openrouter.Client
}

// NewOpenRouterProvider wraps an OpenRouter client.
func NewOpenRouterProvider(client *openrouter.Client) *OpenRouterProvider {
	return &OpenRouterProvider{client: client}
}

func (p *OpenRouterProvider) Name() ProviderName { return ProviderOpenRouter }

func (p *OpenRouterProvider) clientForOpts(opt ChatOptions) *openrouter.Client {
	if p == nil || p.client == nil {
		return nil
	}
	if opt.Timeout > 0 {
		return p.client.WithTimeout(opt.Timeout)
	}
	return p.client
}

func (p *OpenRouterProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	opt := firstChatOptions(opts)
	client := p.clientForOpts(opt)
	if client == nil {
		return ChatResult{}, fmt.Errorf("aiprovider: openrouter not configured")
	}
	msgs := toOpenRouterMessages(messages)
	orOpt := openrouter.ChatOptions{JSONMode: opt.JSONMode, MaxTokens: opt.MaxTokens}
	got, err := client.ChatCompletion(modelID, msgs, orOpt)
	if err != nil {
		return ChatResult{}, err
	}
	return fromOpenRouterResult(got), nil
}

func (p *OpenRouterProvider) CompleteStream(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
	opt := firstChatOptions(opts)
	client := p.clientForOpts(opt)
	if client == nil {
		return ChatResult{}, fmt.Errorf("aiprovider: openrouter not configured")
	}
	msgs := toOpenRouterMessages(messages)
	wrapped := onChunk
	if ctx != nil {
		wrapped = func(text string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if onChunk == nil {
				return nil
			}
			return onChunk(text)
		}
	}
	got, err := client.ChatCompletionStream(modelID, msgs, openrouter.ChunkHandler(wrapped))
	if err != nil {
		return ChatResult{}, err
	}
	return fromOpenRouterResult(got), nil
}

func (p *OpenRouterProvider) CompleteVision(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	opt := firstChatOptions(opts)
	client := p.clientForOpts(opt)
	if client == nil {
		return ChatResult{}, fmt.Errorf("aiprovider: openrouter not configured")
	}
	system, userText, imageURLs := splitVisionMessages(messages)
	got, err := client.VisionCompletionMulti(modelID, system, userText, imageURLs, opt.JSONMode)
	if err != nil {
		return ChatResult{}, err
	}
	return fromOpenRouterResult(got), nil
}

func (p *OpenRouterProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, notSupported("embed")
}

// GenerateImage calls OpenRouter's OpenAI-compatible images API.
func (p *OpenRouterProvider) GenerateImage(ctx context.Context, modelID string, prompt string, opts ...ImageOptions) (ImageResult, error) {
	if p == nil || p.client == nil {
		return ImageResult{}, fmt.Errorf("aiprovider: openrouter not configured")
	}
	opt := firstImageOptions(opts)
	apiKey, baseURL, httpCl := p.client.HTTPConfig()
	if strings.TrimSpace(apiKey) == "" {
		return ImageResult{}, fmt.Errorf("aiprovider: openrouter missing API key")
	}
	hc := &httpClient{
		http:    httpCl,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
	if hc.http == nil {
		hc = newHTTPClient(apiKey, baseURL, nil)
	}
	body := map[string]any{
		"model":  modelID,
		"prompt": prompt,
	}
	if opt.N > 0 {
		body["n"] = opt.N
	}
	if opt.Size != "" {
		body["size"] = opt.Size
	}
	b, _, err := hc.postJSON(ctx, ProviderOpenRouter, "/images/generations", body)
	if err != nil {
		return ImageResult{}, err
	}
	var parsed struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ImageResult{}, fmt.Errorf("aiprovider: parse openrouter image response: %w", err)
	}
	out := ImageResult{}
	for _, d := range parsed.Data {
		if d.URL != "" {
			out.URLs = append(out.URLs, d.URL)
		}
		if d.B64JSON != "" {
			out.B64JSON = append(out.B64JSON, d.B64JSON)
		}
	}
	return out, nil
}

func fromOpenRouterResult(got openrouter.ChatResult) ChatResult {
	return ChatResult{
		Text: got.Text,
		Usage: UsageInfo{
			PromptTokens:     got.Usage.PromptTokens,
			CompletionTokens: got.Usage.CompletionTokens,
			TotalTokens:      got.Usage.TotalTokens,
			CostUSD:          got.Usage.CostUSD,
		},
	}
}

func toOpenRouterMessages(messages []Message) []openrouter.Message {
	msgs := make([]openrouter.Message, len(messages))
	for i, m := range messages {
		msgs[i] = openrouter.Message{Role: m.Role, Content: m.TextContent()}
	}
	return msgs
}

func splitVisionMessages(messages []Message) (system, userText string, imageURLs []string) {
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = m.TextContent()
		default:
			if t := m.TextContent(); t != "" {
				if userText == "" {
					userText = t
				} else {
					userText = userText + "\n" + t
				}
			}
			imageURLs = append(imageURLs, m.ImageURLs()...)
		}
	}
	return system, userText, imageURLs
}
