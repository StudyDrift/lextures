package aiprovider

import (
	"context"
	"errors"
)

// Provider is the AI backend abstraction (plans 16.7 FR-1, AP.1 FR-1).
type Provider interface {
	Name() ProviderName
	Complete(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error)
	CompleteStream(ctx context.Context, modelID string, messages []Message, onChunk ChunkHandler, opts ...ChatOptions) (ChatResult, error)
	CompleteVision(ctx context.Context, modelID string, messages []Message, opts ...ChatOptions) (ChatResult, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}

// ImageProvider is an optional capability for image generation (AP.1 FR-1).
type ImageProvider interface {
	GenerateImage(ctx context.Context, modelID string, prompt string, opts ...ImageOptions) (ImageResult, error)
}

// ChunkHandler receives each streamed content token. Returning a non-nil error stops streaming.
type ChunkHandler func(text string) error

// ErrNotSupported indicates the provider does not implement a requested capability.
var ErrNotSupported = errors.New("aiprovider: not supported")

// notSupported wraps ErrNotSupported with a capability name for diagnostics.
func notSupported(capability string) error {
	return errors.Join(ErrNotSupported, errors.New("aiprovider: "+capability+" not supported"))
}

// Capabilities returns the documented capability matrix for a provider name.
func Capabilities(name ProviderName) CapabilitySet {
	switch name {
	case ProviderOpenRouter:
		return CapabilitySet{Complete: true, Stream: true, Vision: true, Embed: false, Image: true}
	case ProviderAnthropic:
		return CapabilitySet{Complete: true, Stream: false, Vision: true, Embed: false, Image: false}
	case ProviderOpenAI, ProviderAzureOpenAI:
		return CapabilitySet{Complete: true, Stream: true, Vision: true, Embed: true, Image: false}
	case ProviderBedrock, ProviderVertex:
		return CapabilitySet{Complete: true, Stream: false, Vision: false, Embed: false, Image: false}
	case ProviderDryRun:
		return CapabilitySet{Complete: true, Stream: true, Vision: true, Embed: true, Image: true}
	default:
		return CapabilitySet{}
	}
}

// VisionMessages builds a system + user multimodal message list for CompleteVision.
func VisionMessages(systemPrompt, userText string, imageURLs []string) []Message {
	userParts := make([]ContentPart, 0, 1+len(imageURLs))
	if userText != "" {
		userParts = append(userParts, ContentPart{Type: ContentPartText, Text: userText})
	}
	for _, u := range imageURLs {
		if u == "" {
			continue
		}
		userParts = append(userParts, ContentPart{Type: ContentPartImageURL, ImageURL: u})
	}
	out := make([]Message, 0, 2)
	if systemPrompt != "" {
		out = append(out, Message{Role: "system", Content: systemPrompt})
	}
	out = append(out, Message{Role: "user", Parts: userParts})
	return out
}

// TextContent returns the effective text for a message (Parts text joined, else Content).
func (m Message) TextContent() string {
	if len(m.Parts) == 0 {
		return m.Content
	}
	var out string
	for _, p := range m.Parts {
		if p.Type == ContentPartText || (p.Type == "" && p.Text != "") {
			out += p.Text
		}
	}
	if out != "" {
		return out
	}
	return m.Content
}

// ImageURLs returns image URL parts from a message.
func (m Message) ImageURLs() []string {
	var urls []string
	for _, p := range m.Parts {
		if p.Type == ContentPartImageURL && p.ImageURL != "" {
			urls = append(urls, p.ImageURL)
		}
	}
	return urls
}
