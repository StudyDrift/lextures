package aiprovider

import "context"

// DryRunProvider returns synthetic responses without calling any backend (FR-9).
type DryRunProvider struct{}

func (p *DryRunProvider) Name() ProviderName { return ProviderDryRun }

func (p *DryRunProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = opts
	var last string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			last = messages[i].Content
			break
		}
	}
	return ChatResult{
		Text: "Dry-run response for: " + last,
		Usage: UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

func (p *DryRunProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	return []float32{0.1, 0.2, 0.3}, nil
}