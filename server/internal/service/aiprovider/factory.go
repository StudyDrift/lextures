package aiprovider

import (
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/openrouter"
)

// Factory builds concrete providers from configuration.
type Factory struct {
	PlatformOpenRouter *openrouter.Client
}

// Build constructs a provider for the given name, API key, and optional settings JSON.
func (f *Factory) Build(name ProviderName, apiKey string, extra map[string]any) (Provider, error) {
	switch name {
	case ProviderOpenRouter:
		if f != nil && f.PlatformOpenRouter != nil {
			return NewOpenRouterProvider(f.PlatformOpenRouter), nil
		}
		if strings.TrimSpace(apiKey) != "" {
			return NewOpenRouterProvider(openrouter.NewClient(apiKey)), nil
		}
		return nil, fmt.Errorf("aiprovider: openrouter not configured")
	case ProviderAnthropic:
		return NewAnthropicProvider(apiKey), nil
	case ProviderOpenAI:
		return NewOpenAIProvider(apiKey), nil
	case ProviderAzureOpenAI:
		base := stringSetting(extra, "azure_base_url")
		if base == "" {
			return nil, fmt.Errorf("aiprovider: azure_openai requires azure_base_url in settings")
		}
		return NewAzureOpenAIProvider(apiKey, base), nil
	case ProviderBedrock:
		base := stringSetting(extra, "bedrock_base_url")
		if base == "" {
			region := stringSetting(extra, "aws_region")
			if region == "" {
				region = "us-east-1"
			}
			base = "https://bedrock-runtime." + region + ".amazonaws.com"
		}
		return NewBedrockProvider(apiKey, base), nil
	case ProviderVertex:
		base := stringSetting(extra, "vertex_base_url")
		if base == "" {
			project := stringSetting(extra, "gcp_project")
			location := stringSetting(extra, "gcp_location")
			if project == "" || location == "" {
				return nil, fmt.Errorf("aiprovider: vertex requires vertex_base_url or gcp_project+gcp_location")
			}
			base = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models",
				location, project, location)
		}
		return NewVertexProvider(apiKey, base), nil
	case ProviderDryRun:
		return &DryRunProvider{}, nil
	default:
		return nil, fmt.Errorf("aiprovider: unknown provider %q", name)
	}
}

func stringSetting(extra map[string]any, key string) string {
	if extra == nil {
		return ""
	}
	v, ok := extra[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}