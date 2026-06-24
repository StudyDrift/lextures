// Package aiprovider defines a provider-agnostic AI abstraction (plan 16.7).
package aiprovider

import "time"

// ProviderName identifies a backend implementation.
type ProviderName string

const (
	ProviderOpenRouter ProviderName = "openrouter"
	ProviderAnthropic  ProviderName = "anthropic"
	ProviderOpenAI     ProviderName = "openai"
	ProviderAzureOpenAI ProviderName = "azure_openai"
	ProviderBedrock    ProviderName = "bedrock"
	ProviderVertex     ProviderName = "vertex"
	ProviderDryRun     ProviderName = "dry_run"
)

// Message is one chat turn.
type Message struct {
	Role    string
	Content string
}

// ChatOptions configures optional completion behavior.
type ChatOptions struct {
	JSONMode bool
}

// UsageInfo is token and cost metadata from a provider response.
type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
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

// CallMeta describes which provider answered a request.
type CallMeta struct {
	Provider   ProviderName
	ModelAlias string
	ModelID    string
	Latency    time.Duration
	Usage      UsageInfo
}

// Settings holds per-tenant provider configuration (non-secret fields).
type Settings struct {
	Provider         ProviderName
	ModelAlias       string
	FallbackProvider *ProviderName
	BYOKConfigured   bool
	Extra            map[string]any
}