// Package aiprovider defines a provider-agnostic AI abstraction (plans 16.7, AP.1).
//
// # Capability matrix
//
//	| Provider      | Complete | Stream | Vision | Embed | Image |
//	|---------------|----------|--------|--------|-------|-------|
//	| openrouter    | yes      | yes    | yes    | no    | yes   |
//	| anthropic     | yes      | no     | yes    | no    | no    |
//	| openai        | yes      | yes    | yes    | yes   | no    |
//	| azure_openai  | yes      | yes    | yes    | yes   | no    |
//	| bedrock       | yes      | no     | no     | no    | no    |
//	| vertex        | yes      | no     | no     | no    | no    |
//	| dry_run       | yes      | yes    | yes    | yes   | yes   |
//
// Unimplemented capabilities return ErrNotSupported (errors.Is). Adding a
// provider means implementing Provider and, optionally, ImageProvider.
package aiprovider

import "time"

// ProviderName identifies a backend implementation.
type ProviderName string

const (
	ProviderOpenRouter  ProviderName = "openrouter"
	ProviderAnthropic   ProviderName = "anthropic"
	ProviderOpenAI      ProviderName = "openai"
	ProviderAzureOpenAI ProviderName = "azure_openai"
	ProviderBedrock     ProviderName = "bedrock"
	ProviderVertex      ProviderName = "vertex"
	ProviderDryRun      ProviderName = "dry_run"
)

// ContentPartType discriminates multimodal message segments.
type ContentPartType string

const (
	ContentPartText     ContentPartType = "text"
	ContentPartImageURL ContentPartType = "image_url"
)

// ContentPart is a multimodal message segment (text or image URL / data-URL).
type ContentPart struct {
	Type     ContentPartType
	Text     string
	ImageURL string
}

// Message is one chat turn. When Parts is non-empty it takes precedence over Content.
type Message struct {
	Role    string
	Content string
	Parts   []ContentPart
}

// ChatOptions configures optional completion behavior (parity with openrouter.ChatOptions + timeout).
type ChatOptions struct {
	JSONMode    bool
	MaxTokens   int
	Temperature *float64
	// Timeout overrides the default HTTP client timeout when > 0.
	Timeout time.Duration
}

// UsageInfo is token and cost metadata from a provider response.
type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
	// CostEstimated is true when CostUSD came from the local price table (AP.6).
	CostEstimated bool
}

// HasData reports whether the provider returned any usage metadata.
func (u UsageInfo) HasData() bool {
	return u.TotalTokens > 0 || u.PromptTokens > 0 || u.CompletionTokens > 0 || u.CostUSD > 0
}

// ChatResult is assistant text plus optional usage metadata.
type ChatResult struct {
	Text  string
	Usage UsageInfo
}

// ImageOptions configures optional image generation.
type ImageOptions struct {
	N    int    // number of images; providers may clamp to 1
	Size string // e.g. "1024x1024"
}

// ImageResult holds generated image URLs and/or base64 payloads.
type ImageResult struct {
	URLs    []string
	B64JSON []string
	Usage   UsageInfo
}

// CallMeta describes which provider answered a request.
type CallMeta struct {
	Provider   ProviderName
	ModelAlias string
	ModelID    string
	Latency    time.Duration
	Usage      UsageInfo
	Operation  string // complete|stream|vision|embed|image
	AuthMode   string // api_key|access_key|iam_role|service_account|adc (AP.8)
}

// Settings holds per-tenant provider configuration (non-secret fields).
type Settings struct {
	Provider         ProviderName
	ModelAlias       string
	FallbackProvider *ProviderName
	BYOKConfigured   bool
	Extra            map[string]any
}

// CapabilitySet describes which operations a provider supports.
type CapabilitySet struct {
	Complete bool
	Stream   bool
	Vision   bool
	Embed    bool
	Image    bool
}

// Operation labels for metrics (AP.1 observability).
const (
	OpComplete = "complete"
	OpStream   = "stream"
	OpVision   = "vision"
	OpEmbed    = "embed"
	OpImage    = "image"
)
