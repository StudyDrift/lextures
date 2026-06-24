package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
)

const anthropicDefaultBase = "https://api.anthropic.com"

// AnthropicProvider calls the Anthropic Messages API.
type AnthropicProvider struct {
	client *httpClient
}

// NewAnthropicProvider builds an Anthropic-direct provider.
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		client: newHTTPClient(apiKey, anthropicDefaultBase, map[string]string{
			"anthropic-version": "2023-06-01",
			"x-api-key":         apiKey,
		}),
	}
}

// NewAnthropicProviderWithBaseURL is for tests.
func NewAnthropicProviderWithBaseURL(apiKey, baseURL string) *AnthropicProvider {
	return &AnthropicProvider{
		client: newHTTPClient(apiKey, baseURL, map[string]string{
			"anthropic-version": "2023-06-01",
			"x-api-key":         apiKey,
		}),
	}
}

func (p *AnthropicProvider) Name() ProviderName { return ProviderAnthropic }

func (p *AnthropicProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.apiKey == "" {
		return ChatResult{}, fmt.Errorf("aiprovider: anthropic not configured")
	}
	var system string
	var msgs []map[string]string
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = m.Content
		default:
			msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
		}
	}
	body := map[string]any{
		"model":      modelID,
		"max_tokens": 4096,
		"messages":   msgs,
	}
	if system != "" {
		body["system"] = system
	}
	var opt ChatOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	b, _, err := p.client.postJSON(ctx, ProviderAnthropic, "/v1/messages", body)
	if err != nil {
		return ChatResult{}, err
	}
	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ChatResult{}, fmt.Errorf("aiprovider: parse anthropic response: %w", err)
	}
	var text string
	for _, block := range parsed.Content {
		if block.Text != "" {
			text = block.Text
			break
		}
	}
	return ChatResult{
		Text: text,
		Usage: UsageInfo{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}, nil
}

func (p *AnthropicProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, ErrNotSupported
}