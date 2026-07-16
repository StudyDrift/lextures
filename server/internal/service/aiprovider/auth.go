package aiprovider

import (
	"fmt"
	"strings"
)

// Auth mode constants (AP.8 FR-2 / FR-3).
const (
	AuthModeAPIKey         = "api_key"
	AuthModeAccessKey      = "access_key" // Bedrock static AWS keys
	AuthModeIAMRole        = "iam_role"   // Bedrock default credential chain
	AuthModeServiceAccount = "service_account"
	AuthModeADC            = "adc"

	defaultAzureAPIVersion = "2024-10-21"

	// Secret key names (mirror aiprovidercreds; kept here to avoid tight coupling in Build).
	secretKeyAPIKey               = "api_key"
	secretKeyAWSAccessKeyID       = "aws_access_key_id"
	secretKeyAWSSecretAccessKey   = "aws_secret_access_key"
	secretKeyServiceAccountJSON   = "service_account_json"
)

// AuthMaterial holds decrypted secrets for building a provider (AP.8 FR-6).
// Never log or serialize plaintext values.
type AuthMaterial struct {
	APIKey  string
	Secrets map[string]string
}

// Secret returns a named secret. Falls back to APIKey for secret_key=api_key.
func (m AuthMaterial) Secret(key string) string {
	key = strings.TrimSpace(key)
	if m.Secrets != nil {
		if v := strings.TrimSpace(m.Secrets[key]); v != "" {
			return v
		}
	}
	if key == secretKeyAPIKey {
		return strings.TrimSpace(m.APIKey)
	}
	return ""
}

// AuthModeFromSettings reads settings.auth_mode with provider-specific defaults.
func AuthModeFromSettings(provider ProviderName, extra map[string]any) string {
	mode := strings.ToLower(strings.TrimSpace(stringSetting(extra, "auth_mode")))
	if mode != "" {
		return mode
	}
	switch provider {
	case ProviderBedrock, ProviderVertex, ProviderAzureOpenAI:
		return AuthModeAPIKey
	default:
		return AuthModeAPIKey
	}
}

// ResolveAzureDeployment maps a model alias/id to an Azure deployment name (AP.8 AC-1).
func ResolveAzureDeployment(modelID string, extra map[string]any) string {
	modelID = strings.TrimSpace(modelID)
	if dep := deploymentMapLookup(extra, modelID); dep != "" {
		return dep
	}
	if def := stringSetting(extra, "default_deployment"); def != "" {
		return def
	}
	return modelID
}

func deploymentMapLookup(extra map[string]any, modelID string) string {
	if extra == nil {
		return ""
	}
	raw, ok := extra["deployments"]
	if !ok || raw == nil {
		return ""
	}
	switch m := raw.(type) {
	case map[string]any:
		if v, ok := m[modelID]; ok {
			return strings.TrimSpace(fmt.Sprint(v))
		}
	case map[string]string:
		return strings.TrimSpace(m[modelID])
	}
	return ""
}

func validateAzureSettings(extra map[string]any) error {
	if stringSetting(extra, "azure_base_url") == "" {
		return newConfigError(ProviderAzureOpenAI, "azure_openai requires azure_base_url in settings")
	}
	return nil
}

func validateBedrockSettings(mode string, auth AuthMaterial) error {
	switch mode {
	case AuthModeAPIKey:
		// Bearer/gateway key optional at build time; Complete will fail if missing when needed.
		return nil
	case AuthModeAccessKey:
		if auth.Secret(secretKeyAWSAccessKeyID) == "" {
			return newConfigError(ProviderBedrock, "bedrock auth_mode=access_key requires aws_access_key_id secret")
		}
		if auth.Secret(secretKeyAWSSecretAccessKey) == "" {
			return newConfigError(ProviderBedrock, "bedrock auth_mode=access_key requires aws_secret_access_key secret")
		}
		return nil
	case AuthModeIAMRole:
		return nil
	default:
		return newConfigError(ProviderBedrock, fmt.Sprintf("bedrock unsupported auth_mode %q (use api_key, access_key, or iam_role)", mode))
	}
}

func validateVertexSettings(mode string, auth AuthMaterial, extra map[string]any) error {
	base := stringSetting(extra, "vertex_base_url")
	project := stringSetting(extra, "gcp_project")
	location := stringSetting(extra, "gcp_location")
	if base == "" && (project == "" || location == "") {
		return newConfigError(ProviderVertex, "vertex requires vertex_base_url or gcp_project+gcp_location")
	}
	switch mode {
	case AuthModeAPIKey:
		if auth.Secret(secretKeyAPIKey) == "" {
			return newConfigError(ProviderVertex, "vertex auth_mode=api_key requires an API key")
		}
		return nil
	case AuthModeServiceAccount:
		if auth.Secret(secretKeyServiceAccountJSON) == "" {
			return newConfigError(ProviderVertex, "vertex auth_mode=service_account requires service_account_json secret")
		}
		return nil
	case AuthModeADC:
		return nil
	default:
		return newConfigError(ProviderVertex, fmt.Sprintf("vertex unsupported auth_mode %q (use api_key, service_account, or adc)", mode))
	}
}

func bedrockRegion(extra map[string]any) string {
	region := stringSetting(extra, "aws_region")
	if region == "" {
		return "us-east-1"
	}
	return region
}

func bedrockBaseURL(extra map[string]any) string {
	base := stringSetting(extra, "bedrock_base_url")
	if base != "" {
		return base
	}
	return "https://bedrock-runtime." + bedrockRegion(extra) + ".amazonaws.com"
}

func vertexBaseURL(extra map[string]any) (string, error) {
	base := stringSetting(extra, "vertex_base_url")
	if base != "" {
		return base, nil
	}
	project := stringSetting(extra, "gcp_project")
	location := stringSetting(extra, "gcp_location")
	if project == "" || location == "" {
		return "", newConfigError(ProviderVertex, "vertex requires vertex_base_url or gcp_project+gcp_location")
	}
	return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models",
		location, project, location), nil
}

func azureAPIVersion(extra map[string]any) string {
	v := stringSetting(extra, "azure_api_version")
	if v == "" {
		return defaultAzureAPIVersion
	}
	return v
}
