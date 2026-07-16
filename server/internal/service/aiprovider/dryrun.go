package aiprovider

import (
	"context"
	"strings"
)

// DryRunProvider returns synthetic responses without calling any backend (FR-9 / AP.1 FR-9).
type DryRunProvider struct{}

func (p *DryRunProvider) Name() ProviderName { return ProviderDryRun }

func (p *DryRunProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = opts
	return ChatResult{
		Text:  "Dry-run response for: " + lastUserText(messages),
		Usage: dryRunUsage(),
	}, nil
}

func (p *DryRunProvider) CompleteStream(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = opts
	parts := []string{"Dry-run ", "stream ", "for: ", lastUserText(messages)}
	var sb strings.Builder
	for _, part := range parts {
		sb.WriteString(part)
		if onChunk != nil {
			if err := onChunk(part); err != nil {
				return ChatResult{Text: sb.String(), Usage: dryRunUsage()}, err
			}
		}
	}
	return ChatResult{Text: sb.String(), Usage: dryRunUsage()}, nil
}

func (p *DryRunProvider) CompleteVision(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = opts
	return ChatResult{
		Text:  "Dry-run vision response for: " + lastUserText(messages),
		Usage: dryRunUsage(),
	}, nil
}

func (p *DryRunProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return []float32{0.1, 0.2, 0.3}, nil
}

func (p *DryRunProvider) GenerateImage(ctx context.Context, modelID string, prompt string, opts ...ImageOptions) (ImageResult, error) {
	_ = ctx
	_ = modelID
	_ = opts
	return ImageResult{
		URLs:  []string{"https://example.invalid/dry-run.png"},
		Usage: dryRunUsage(),
	}, nil
}

func lastUserText(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].TextContent()
		}
	}
	return ""
}

func dryRunUsage() UsageInfo {
	return UsageInfo{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
}
