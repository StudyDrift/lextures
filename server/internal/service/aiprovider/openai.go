package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
)

const openAIDefaultBase = "https://api.openai.com/v1"

// OpenAIProvider calls the OpenAI Chat Completions API (Azure-compatible via custom base URL).
type OpenAIProvider struct {
	name   ProviderName
	client *httpClient
}

// NewOpenAIProvider builds an OpenAI-direct provider.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		name:   ProviderOpenAI,
		client: newHTTPClient(apiKey, openAIDefaultBase, nil),
	}
}

// NewAzureOpenAIProvider builds an Azure OpenAI provider.
func NewAzureOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	return &OpenAIProvider{
		name:   ProviderAzureOpenAI,
		client: newHTTPClient(apiKey, baseURL, map[string]string{"api-key": apiKey}),
	}
}

// NewOpenAIProviderWithBaseURL is for tests.
func NewOpenAIProviderWithBaseURL(apiKey, baseURL string) *OpenAIProvider {
	return &OpenAIProvider{
		name:   ProviderOpenAI,
		client: newHTTPClient(apiKey, baseURL, nil),
	}
}

func (p *OpenAIProvider) Name() ProviderName {
	if p != nil && p.name != "" {
		return p.name
	}
	return ProviderOpenAI
}

func (p *OpenAIProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.apiKey == "" {
		return ChatResult{}, fmt.Errorf("aiprovider: openai not configured")
	}
	msgs := make([]map[string]string, len(messages))
	for i, m := range messages {
		msgs[i] = map[string]string{"role": m.Role, "content": m.Content}
	}
	body := map[string]any{
		"model":    modelID,
		"messages": msgs,
	}
	var opt ChatOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	b, _, err := p.client.postJSON(ctx, p.Name(), "/chat/completions", body)
	if err != nil {
		return ChatResult{}, err
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content *string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ChatResult{}, fmt.Errorf("aiprovider: parse openai response: %w", err)
	}
	out := ChatResult{
		Usage: UsageInfo{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}
	if len(parsed.Choices) > 0 && parsed.Choices[0].Message.Content != nil {
		out.Text = *parsed.Choices[0].Message.Content
	}
	return out, nil
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, ErrNotSupported
}