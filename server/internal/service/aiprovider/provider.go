package aiprovider

import "context"

// Provider is the AI backend abstraction (plan 16.7 FR-1).
type Provider interface {
	Name() ProviderName
	Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}

// ErrNotSupported indicates the provider does not implement embeddings.
var ErrNotSupported = errNotSupported{}

type errNotSupported struct{}

func (errNotSupported) Error() string { return "aiprovider: embeddings not supported" }