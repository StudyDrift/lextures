package aiprovider

import "context"

// MockProvider is a test double for Provider (and optionally ImageProvider).
type MockProvider struct {
	NameValue          ProviderName
	CompleteFunc       func(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error)
	CompleteStreamFunc func(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error)
	CompleteVisionFunc func(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error)
	EmbedFunc          func(ctx context.Context, text string) ([]float32, error)
	GenerateImageFunc  func(ctx context.Context, modelID string, prompt string, opts ...ImageOptions) (ImageResult, error)
	CompleteCalls      int
	StreamCalls        int
	VisionCalls        int
}

func (m *MockProvider) Name() ProviderName {
	if m != nil && m.NameValue != "" {
		return m.NameValue
	}
	return ProviderDryRun
}

func (m *MockProvider) Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	m.CompleteCalls++
	if m.CompleteFunc != nil {
		return m.CompleteFunc(ctx, modelID, messages, opts...)
	}
	return ChatResult{}, notSupported("complete")
}

func (m *MockProvider) CompleteStream(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error) {
	m.StreamCalls++
	if m.CompleteStreamFunc != nil {
		return m.CompleteStreamFunc(ctx, modelID, messages, onChunk, opts...)
	}
	return ChatResult{}, notSupported("stream")
}

func (m *MockProvider) CompleteVision(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error) {
	m.VisionCalls++
	if m.CompleteVisionFunc != nil {
		return m.CompleteVisionFunc(ctx, modelID, messages, opts...)
	}
	return ChatResult{}, notSupported("vision")
}

func (m *MockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.EmbedFunc != nil {
		return m.EmbedFunc(ctx, text)
	}
	return nil, notSupported("embed")
}

func (m *MockProvider) GenerateImage(ctx context.Context, modelID string, prompt string, opts ...ImageOptions) (ImageResult, error) {
	if m.GenerateImageFunc != nil {
		return m.GenerateImageFunc(ctx, modelID, prompt, opts...)
	}
	return ImageResult{}, notSupported("image")
}
