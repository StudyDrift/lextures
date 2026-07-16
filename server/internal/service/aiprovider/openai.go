package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const openAIDefaultBase = "https://api.openai.com/v1"

// AzureOptions configures Azure OpenAI deployment routing (AP.8 FR-1).
type AzureOptions struct {
	APIVersion        string
	Deployments       map[string]string
	DefaultDeployment string
}

// OpenAIProvider calls the OpenAI Chat Completions API (Azure-compatible via custom base URL).
type OpenAIProvider struct {
	name              ProviderName
	client            *httpClient
	azureAPIVersion   string
	azureDeployments  map[string]string
	azureDefaultDeploy string
}

// NewOpenAIProvider builds an OpenAI-direct provider.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		name:   ProviderOpenAI,
		client: newHTTPClient(apiKey, openAIDefaultBase, nil),
	}
}

// NewAzureOpenAIProvider builds an Azure OpenAI provider with deployment mapping.
func NewAzureOpenAIProvider(apiKey, baseURL string, opts ...AzureOptions) *OpenAIProvider {
	var o AzureOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if strings.TrimSpace(o.APIVersion) == "" {
		o.APIVersion = defaultAzureAPIVersion
	}
	return &OpenAIProvider{
		name:               ProviderAzureOpenAI,
		client:             newHTTPClient(apiKey, baseURL, map[string]string{"api-key": apiKey}),
		azureAPIVersion:    o.APIVersion,
		azureDeployments:   o.Deployments,
		azureDefaultDeploy: o.DefaultDeployment,
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

func (p *OpenAIProvider) isAzure() bool {
	return p != nil && p.name == ProviderAzureOpenAI
}

func (p *OpenAIProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.apiKey == "" {
		return ChatResult{}, newConfigError(p.Name(), "openai not configured")
	}
	opt := firstChatOptions(opts)
	ctx, cancel := withChatTimeout(ctx, opt)
	defer cancel()
	client := p.client.withTimeout(opt.Timeout)

	path, body := p.chatRequest(modelID, messages, opt, false)
	b, _, err := client.postJSON(ctx, p.Name(), path, body)
	if err != nil {
		return ChatResult{}, err
	}
	return parseOpenAIChatResponse(b)
}

func (p *OpenAIProvider) CompleteStream(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.apiKey == "" {
		return ChatResult{}, newConfigError(p.Name(), "openai not configured")
	}
	opt := firstChatOptions(opts)
	ctx, cancel := withChatTimeout(ctx, opt)
	defer cancel()
	client := p.client.withTimeout(opt.Timeout)

	path, body := p.chatRequest(modelID, messages, opt, true)
	return streamOpenAICompatible(ctx, client, p.Name(), path, body, onChunk)
}

func (p *OpenAIProvider) CompleteVision(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.apiKey == "" {
		return ChatResult{}, newConfigError(p.Name(), "openai not configured")
	}
	opt := firstChatOptions(opts)
	ctx, cancel := withChatTimeout(ctx, opt)
	defer cancel()
	client := p.client.withTimeout(opt.Timeout)

	path, body := p.chatRequest(modelID, messages, opt, false)
	b, _, err := client.postJSON(ctx, p.Name(), path, body)
	if err != nil {
		return ChatResult{}, err
	}
	return parseOpenAIChatResponse(b)
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if p == nil || p.client == nil || p.client.apiKey == "" {
		return nil, newConfigError(p.Name(), "openai not configured")
	}
	body := map[string]any{
		"model": "text-embedding-3-small",
		"input": text,
	}
	path := "/embeddings"
	if p.isAzure() {
		dep := p.resolveDeployment("text-embedding-3-small")
		path = azureDeploymentPath(dep, "embeddings", p.azureAPIVersion)
		delete(body, "model")
	}
	b, _, err := p.client.postJSON(ctx, p.Name(), path, body)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, fmt.Errorf("aiprovider: parse openai embeddings: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("aiprovider: empty embeddings response")
	}
	return parsed.Data[0].Embedding, nil
}

func (p *OpenAIProvider) chatRequest(modelID string, messages []Message, opt ChatOptions, stream bool) (string, map[string]any) {
	if p.isAzure() {
		dep := p.resolveDeployment(modelID)
		path := azureDeploymentPath(dep, "chat/completions", p.azureAPIVersion)
		body := openAIChatBody("", messages, opt, stream) // empty model → omit for Azure
		delete(body, "model")
		return path, body
	}
	return "/chat/completions", openAIChatBody(modelID, messages, opt, stream)
}

func (p *OpenAIProvider) resolveDeployment(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if p.azureDeployments != nil {
		if d := strings.TrimSpace(p.azureDeployments[modelID]); d != "" {
			return d
		}
	}
	if d := strings.TrimSpace(p.azureDefaultDeploy); d != "" {
		return d
	}
	return modelID
}

func azureDeploymentPath(deployment, operation, apiVersion string) string {
	deployment = strings.Trim(strings.TrimSpace(deployment), "/")
	operation = strings.Trim(strings.TrimSpace(operation), "/")
	q := url.Values{}
	q.Set("api-version", apiVersion)
	return "/openai/deployments/" + url.PathEscape(deployment) + "/" + operation + "?" + q.Encode()
}

func openAIChatBody(modelID string, messages []Message, opt ChatOptions, stream bool) map[string]any {
	msgs := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		msg := map[string]any{"role": m.Role}
		if len(m.Parts) > 0 {
			parts := make([]map[string]any, 0, len(m.Parts))
			for _, part := range m.Parts {
				switch part.Type {
				case ContentPartImageURL:
					parts = append(parts, map[string]any{
						"type":      "image_url",
						"image_url": map[string]string{"url": part.ImageURL},
					})
				default:
					parts = append(parts, map[string]any{"type": "text", "text": part.Text})
				}
			}
			msg["content"] = parts
		} else {
			msg["content"] = m.Content
		}
		msgs = append(msgs, msg)
	}
	body := map[string]any{
		"messages": msgs,
		"stream":   stream,
	}
	if modelID != "" {
		body["model"] = modelID
	}
	if opt.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	if opt.MaxTokens > 0 {
		body["max_tokens"] = opt.MaxTokens
	}
	applyTemperature(body, opt)
	return body
}

func parseOpenAIChatResponse(b []byte) (ChatResult, error) {
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
