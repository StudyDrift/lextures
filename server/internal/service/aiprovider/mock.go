package aiprovider

import "context"

// MockProvider is a test double for Provider.
type MockProvider struct {
	NameValue     ProviderName
	CompleteFunc  func(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error)
	EmbedFunc     func(ctx context.Context, text string) ([]float32, error)
	CompleteCalls int
}

func (m *MockProvider) Name() ProviderName {
	if m.NameValue != "" {
		return m.NameValue
	}
	return ProviderDryRun
}

func (m *MockProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	m.CompleteCalls++
	if m.CompleteFunc != nil {
		return m.CompleteFunc(ctx, modelID, messages, opts...)
	}
	return ChatResult{Text: "mock"}, nil
}

func (m *MockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.EmbedFunc != nil {
		return m.EmbedFunc(ctx, text)
	}
	return []float32{1}, nil
}