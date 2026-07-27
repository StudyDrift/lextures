package aiprovider

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
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

// CompleteWithTools implements a minimal tool-calling loop for dry-run (CT.6).
// If tools are provided, it invokes the first tool with empty/default args once, then answers.
func (p *DryRunProvider) CompleteWithTools(
	ctx context.Context,
	modelID string,
	messages []Message,
	tools []ToolSpec,
	handler ToolHandler,
	opts ...ChatOptions,
) (ChatResult, error) {
	_ = ctx
	_ = modelID
	_ = opts
	if len(tools) > 0 && handler != nil {
		args := `{}`
		if tools[0].Name == "fetch_link" {
			// Prefer a URL mentioned in the latest user message when present.
			args = `{"url":"https://example.com"}`
			for i := len(messages) - 1; i >= 0; i-- {
				if messages[i].Role != "user" {
					continue
				}
				text := messages[i].TextContent()
				if idx := strings.Index(text, "https://"); idx >= 0 {
					u := text[idx:]
					if sp := strings.IndexAny(u, " \n\t"); sp > 0 {
						u = u[:sp]
					}
					b, _ := json.Marshal(map[string]string{"url": u})
					args = string(b)
					break
				}
			}
		}
		_, _ = handler(ToolCall{ID: "dry-run-1", Name: tools[0].Name, Arguments: args})
	}
	return ChatResult{
		Text:  "Dry-run tool response for: " + lastUserText(messages),
		Usage: dryRunUsage(),
	}, nil
}

// DryRunToolCallingCompleter adapts DryRunProvider to the tenant-aware ToolCallingCompleter.
type DryRunToolCallingCompleter struct{}

func (DryRunToolCallingCompleter) Complete(ctx context.Context, orgID *uuid.UUID, modelOverride string, messages []Message, opts ...ChatOptions) (ChatResult, CallMeta, error) {
	_ = orgID
	p := &DryRunProvider{}
	got, err := p.Complete(ctx, modelOverride, messages, opts...)
	return got, CallMeta{Provider: ProviderDryRun, ModelAlias: "dry-run", ModelID: "dry-run", Operation: OpComplete, Usage: got.Usage}, err
}

func (DryRunToolCallingCompleter) CompleteWithTools(
	ctx context.Context,
	orgID *uuid.UUID,
	modelOverride string,
	messages []Message,
	tools []ToolSpec,
	handler ToolHandler,
	opts ...ChatOptions,
) (ChatResult, CallMeta, error) {
	_ = orgID
	p := &DryRunProvider{}
	got, err := p.CompleteWithTools(ctx, modelOverride, messages, tools, handler, opts...)
	return got, CallMeta{Provider: ProviderDryRun, ModelAlias: "dry-run", ModelID: "dry-run", Operation: OpComplete, Usage: got.Usage}, err
}
