package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// BedrockProvider calls the AWS Bedrock Converse API via HTTP (base URL includes region endpoint).
type BedrockProvider struct {
	client *httpClient
}

// NewBedrockProvider builds a Bedrock provider. baseURL should be like
// https://bedrock-runtime.us-east-1.amazonaws.com and modelID is included in the path.
func NewBedrockProvider(apiKey, baseURL string) *BedrockProvider {
	headers := map[string]string{}
	if strings.TrimSpace(apiKey) != "" {
		// Support bearer-token gateways and local test servers.
		headers["Authorization"] = "Bearer " + strings.TrimSpace(apiKey)
	}
	return &BedrockProvider{client: newHTTPClient(apiKey, baseURL, headers)}
}

func (p *BedrockProvider) Name() ProviderName { return ProviderBedrock }

func (p *BedrockProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.baseURL == "" {
		return ChatResult{}, fmt.Errorf("aiprovider: bedrock not configured")
	}
	var system []map[string]any
	var msgs []map[string]any
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = append(system, map[string]any{"text": m.Content})
		default:
			msgs = append(msgs, map[string]any{
				"role":    m.Role,
				"content": []map[string]string{{"text": m.Content}},
			})
		}
	}
	body := map[string]any{"messages": msgs}
	if len(system) > 0 {
		body["system"] = system
	}
	_ = opts
	path := "/model/" + modelID + "/converse"
	b, _, err := p.client.postJSON(ctx, ProviderBedrock, path, body)
	if err != nil {
		return ChatResult{}, err
	}
	var parsed struct {
		Output struct {
			Message struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		} `json:"output"`
		Usage struct {
			InputTokens  int `json:"inputTokens"`
			OutputTokens int `json:"outputTokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ChatResult{}, fmt.Errorf("aiprovider: parse bedrock response: %w", err)
	}
	var text string
	for _, block := range parsed.Output.Message.Content {
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

func (p *BedrockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, ErrNotSupported
}