package aiprovider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
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
	opt := firstChatOptions(opts)
	ctx, cancel := withChatTimeout(ctx, opt)
	defer cancel()
	client := p.client.withTimeout(opt.Timeout)

	system, msgs := anthropicTextMessages(messages)
	system = ensureJSONSystem(system, opt.JSONMode)
	body := map[string]any{
		"model":      modelID,
		"max_tokens": effectiveMaxTokens(opt, 4096),
		"messages":   msgs,
	}
	if system != "" {
		body["system"] = system
	}
	applyTemperature(body, opt)
	b, _, err := client.postJSON(ctx, ProviderAnthropic, "/v1/messages", body)
	if err != nil {
		return ChatResult{}, err
	}
	return parseAnthropicResponse(b)
}

func (p *AnthropicProvider) CompleteStream(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = messages
	_ = onChunk
	_ = opts
	return ChatResult{}, notSupported("stream")
}

func (p *AnthropicProvider) CompleteVision(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.apiKey == "" {
		return ChatResult{}, fmt.Errorf("aiprovider: anthropic not configured")
	}
	opt := firstChatOptions(opts)
	ctx, cancel := withChatTimeout(ctx, opt)
	defer cancel()
	client := p.client.withTimeout(opt.Timeout)

	system, msgs := anthropicVisionMessages(messages)
	system = ensureJSONSystem(system, opt.JSONMode)
	body := map[string]any{
		"model":      modelID,
		"max_tokens": effectiveMaxTokens(opt, 4096),
		"messages":   msgs,
	}
	if system != "" {
		body["system"] = system
	}
	applyTemperature(body, opt)
	b, _, err := client.postJSON(ctx, ProviderAnthropic, "/v1/messages", body)
	if err != nil {
		return ChatResult{}, err
	}
	return parseAnthropicResponse(b)
}

func (p *AnthropicProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, notSupported("embed")
}

func anthropicTextMessages(messages []Message) (system string, msgs []map[string]any) {
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = m.TextContent()
		default:
			msgs = append(msgs, map[string]any{"role": m.Role, "content": m.TextContent()})
		}
	}
	return system, msgs
}

func anthropicVisionMessages(messages []Message) (system string, msgs []map[string]any) {
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = m.TextContent()
		default:
			parts := make([]map[string]any, 0)
			if len(m.Parts) == 0 {
				parts = append(parts, map[string]any{"type": "text", "text": m.Content})
			} else {
				for _, part := range m.Parts {
					switch part.Type {
					case ContentPartImageURL:
						parts = append(parts, anthropicImagePart(part.ImageURL))
					default:
						if part.Text != "" {
							parts = append(parts, map[string]any{"type": "text", "text": part.Text})
						}
					}
				}
			}
			msgs = append(msgs, map[string]any{"role": m.Role, "content": parts})
		}
	}
	return system, msgs
}

func anthropicImagePart(imageURL string) map[string]any {
	u := strings.TrimSpace(imageURL)
	if strings.HasPrefix(u, "data:") {
		mediaType, data, ok := parseDataURL(u)
		if ok {
			return map[string]any{
				"type": "image",
				"source": map[string]any{
					"type":       "base64",
					"media_type": mediaType,
					"data":       data,
				},
			}
		}
	}
	return map[string]any{
		"type": "image",
		"source": map[string]any{
			"type": "url",
			"url":  u,
		},
	}
}

func parseDataURL(dataURL string) (mediaType, data string, ok bool) {
	// data:[<mediatype>][;base64],<data>
	if !strings.HasPrefix(dataURL, "data:") {
		return "", "", false
	}
	rest := strings.TrimPrefix(dataURL, "data:")
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return "", "", false
	}
	meta := rest[:comma]
	payload := rest[comma+1:]
	mediaType = "image/png"
	mt, _, _ := strings.Cut(meta, ";")
	if mt != "" {
		mediaType = mt
	}
	if strings.Contains(meta, "base64") {
		if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
			return "", "", false
		}
		return mediaType, payload, true
	}
	return mediaType, base64.StdEncoding.EncodeToString([]byte(payload)), true
}

func parseAnthropicResponse(b []byte) (ChatResult, error) {
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
