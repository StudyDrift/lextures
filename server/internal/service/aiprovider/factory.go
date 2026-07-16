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
	return f.BuildWithAuth(name, AuthMaterial{APIKey: apiKey}, extra)
}

// BuildWithAuth constructs a provider using multi-secret auth material (AP.8).
func (f *Factory) BuildWithAuth(name ProviderName, auth AuthMaterial, extra map[string]any) (Provider, error) {
	apiKey := auth.Secret(secretKeyAPIKey)
	switch name {
	case ProviderOpenRouter:
		if f != nil && f.PlatformOpenRouter != nil {
			return NewOpenRouterProvider(f.PlatformOpenRouter), nil
		}
		if strings.TrimSpace(apiKey) != "" {
			return NewOpenRouterProvider(openrouter.NewClient(apiKey)), nil
		}
		return nil, newConfigError(ProviderOpenRouter, "openrouter not configured")
	case ProviderAnthropic:
		return NewAnthropicProvider(apiKey), nil
	case ProviderOpenAI:
		return NewOpenAIProvider(apiKey), nil
	case ProviderAzureOpenAI:
		if err := validateAzureSettings(extra); err != nil {
			return nil, err
		}
		if strings.TrimSpace(apiKey) == "" {
			return nil, newConfigError(ProviderAzureOpenAI, "azure_openai requires an API key")
		}
		base := stringSetting(extra, "azure_base_url")
		return NewAzureOpenAIProvider(apiKey, base, AzureOptions{
			APIVersion:        azureAPIVersion(extra),
			Deployments:       copyStringMap(extra["deployments"]),
			DefaultDeployment: stringSetting(extra, "default_deployment"),
		}), nil
	case ProviderBedrock:
		mode := AuthModeFromSettings(ProviderBedrock, extra)
		if err := validateBedrockSettings(mode, auth); err != nil {
			return nil, err
		}
		return newBedrockProviderFromAuth(mode, auth, extra)
	case ProviderVertex:
		mode := AuthModeFromSettings(ProviderVertex, extra)
		if err := validateVertexSettings(mode, auth, extra); err != nil {
			return nil, err
		}
		return newVertexProviderFromAuth(mode, auth, extra)
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

func copyStringMap(raw any) map[string]string {
	out := map[string]string{}
	if raw == nil {
		return out
	}
	switch m := raw.(type) {
	case map[string]string:
		for k, v := range m {
			if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
				out[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
	case map[string]any:
		for k, v := range m {
			s := strings.TrimSpace(fmt.Sprint(v))
			if strings.TrimSpace(k) != "" && s != "" {
				out[strings.TrimSpace(k)] = s
			}
		}
	}
	return out
}
