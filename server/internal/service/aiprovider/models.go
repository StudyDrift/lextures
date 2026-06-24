package aiprovider

import "fmt"

// ModelAlias is a stable, provider-agnostic model identifier (FR-6).
type ModelAlias string

const (
	AliasClaude35Sonnet ModelAlias = "claude-3-5-sonnet"
	AliasGPT4o          ModelAlias = "gpt-4o"
	AliasGemini15Pro    ModelAlias = "gemini-1.5-pro"
)

type modelMapping struct {
	OpenRouter string
	Anthropic  string
	OpenAI     string
	AzureOpenAI string
	Bedrock    string
	Vertex     string
}

var modelRegistry = map[ModelAlias]modelMapping{
	AliasClaude35Sonnet: {
		OpenRouter:  "anthropic/claude-3.5-sonnet",
		Anthropic:   "claude-3-5-sonnet-20241022",
		OpenAI:      "gpt-4o",
		AzureOpenAI: "gpt-4o",
		Bedrock:     "anthropic.claude-3-5-sonnet-20241022-v2:0",
		Vertex:      "claude-3-5-sonnet",
	},
	AliasGPT4o: {
		OpenRouter:  "openai/gpt-4o",
		Anthropic:   "claude-3-5-sonnet-20241022",
		OpenAI:      "gpt-4o",
		AzureOpenAI: "gpt-4o",
		Bedrock:     "openai.gpt-4o",
		Vertex:      "gpt-4o",
	},
	AliasGemini15Pro: {
		OpenRouter:  "google/gemini-pro-1.5",
		Anthropic:   "claude-3-5-sonnet-20241022",
		OpenAI:      "gpt-4o",
		AzureOpenAI: "gpt-4o",
		Bedrock:     "amazon.titan-text-premier-v1:0",
		Vertex:      "gemini-1.5-pro",
	},
}

// ResolveModelID maps an alias to a provider-specific model id.
func ResolveModelID(alias string, provider ProviderName) (string, error) {
	a := ModelAlias(alias)
	m, ok := modelRegistry[a]
	if !ok {
		return "", fmt.Errorf("aiprovider: unknown model alias %q", alias)
	}
	switch provider {
	case ProviderOpenRouter:
		return m.OpenRouter, nil
	case ProviderAnthropic:
		return m.Anthropic, nil
	case ProviderOpenAI:
		return m.OpenAI, nil
	case ProviderAzureOpenAI:
		return m.AzureOpenAI, nil
	case ProviderBedrock:
		return m.Bedrock, nil
	case ProviderVertex:
		return m.Vertex, nil
	case ProviderDryRun:
		return string(a), nil
	default:
		return "", fmt.Errorf("aiprovider: unsupported provider %q", provider)
	}
}

// ListModelAliases returns registered stable aliases.
func ListModelAliases() []string {
	out := make([]string, 0, len(modelRegistry))
	for alias := range modelRegistry {
		out = append(out, string(alias))
	}
	return out
}

// ListProviders returns supported provider names for admin UI.
func ListProviders() []ProviderName {
	return []ProviderName{
		ProviderOpenRouter,
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderAzureOpenAI,
		ProviderBedrock,
		ProviderVertex,
	}
}