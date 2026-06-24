package aiprovider

import (
	"context"
	"fmt"

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

func (p *OpenRouterProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	if p == nil || p.client == nil {
		return ChatResult{}, fmt.Errorf("aiprovider: openrouter not configured")
	}
	msgs := make([]openrouter.Message, len(messages))
	for i, m := range messages {
		msgs[i] = openrouter.Message{Role: m.Role, Content: m.Content}
	}
	var opt ChatOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	orOpt := openrouter.ChatOptions{JSONMode: opt.JSONMode}
	got, err := p.client.ChatCompletion(modelID, msgs, orOpt)
	if err != nil {
		return ChatResult{}, err
	}
	return ChatResult{
		Text: got.Text,
		Usage: UsageInfo{
			PromptTokens:     got.Usage.PromptTokens,
			CompletionTokens: got.Usage.CompletionTokens,
			TotalTokens:      got.Usage.TotalTokens,
			CostUSD:          got.Usage.CostUSD,
		},
	}, nil
}

func (p *OpenRouterProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return nil, ErrNotSupported
}