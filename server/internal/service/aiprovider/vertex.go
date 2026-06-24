package aiprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// VertexProvider calls Google Vertex AI Gemini generateContent.
type VertexProvider struct {
	client *httpClient
}

// NewVertexProvider builds a Vertex provider. baseURL should include project/location, e.g.
// https://us-central1-aiplatform.googleapis.com/v1/projects/my-proj/locations/us-central1/publishers/google/models
func NewVertexProvider(apiKey, baseURL string) *VertexProvider {
	return &VertexProvider{client: newHTTPClient(apiKey, baseURL, nil)}
}

func (p *VertexProvider) Name() ProviderName { return ProviderVertex }

func (p *VertexProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	if p == nil || p.client == nil || p.client.baseURL == "" {
		return ChatResult{}, fmt.Errorf("aiprovider: vertex not configured")
	}
	var system string
	var contents []map[string]any
	for _, m := range messages {
		switch m.Role {
		case "system":
			system = m.Content
		case "assistant":
			contents = append(contents, map[string]any{
				"role": "model",
				"parts": []map[string]string{{"text": m.Content}},
			})
		default:
			contents = append(contents, map[string]any{
				"role": "user",
				"parts": []map[string]string{{"text": m.Content}},
			})
		}
	}
	body := map[string]any{"contents": contents}
	if system != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": system}},
		}
	}
	_ = opts
	path := "/" + strings.TrimPrefix(modelID, "/") + ":generateContent"
	b, _, err := p.client.postJSON(ctx, ProviderVertex, path, body)
	if err != nil {
		return ChatResult{}, err
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return ChatResult{}, fmt.Errorf("aiprovider: parse vertex response: %w", err)
	}
	var text string
	if len(parsed.Candidates) > 0 {
		for _, part := range parsed.Candidates[0].Content.Parts {
			if part.Text != "" {
				text = part.Text
				break
			}
		}
	}
	total := parsed.UsageMetadata.TotalTokenCount
	if total == 0 {
		total = parsed.UsageMetadata.PromptTokenCount + parsed.UsageMetadata.CandidatesTokenCount
	}
	return ChatResult{
		Text: text,
		Usage: UsageInfo{
			PromptTokens:     parsed.UsageMetadata.PromptTokenCount,
			CompletionTokens: parsed.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      total,
		},
	}, nil
}

func (p *VertexProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, ErrNotSupported
}