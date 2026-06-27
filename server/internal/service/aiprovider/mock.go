package aiprovider

import "context"

// MockProvider is a test double for Provider.
type MockProvider struct {
	NameValue     ProviderName
	CompleteFunc  func(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error)
	EmbedFunc     func(ctx context.Context, text string) ([]float32, error)
	CompleteCalls int
}
