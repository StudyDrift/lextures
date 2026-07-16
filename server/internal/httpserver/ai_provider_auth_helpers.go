package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/aiprovidercreds"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// applyProviderSecrets writes/clears multi-secret material from a PUT body (AP.8 FR-6).
// secrets maps secret_key → plaintext (omit unchanged). clearSecrets lists keys to delete.
func applyProviderSecrets(
	ctx context.Context,
	pool *pgxpool.Pool,
	scope string,
	orgID *uuid.UUID,
	provider string,
	secretsKey []byte,
	secrets map[string]string,
	clearSecrets []string,
) error {
	for _, key := range clearSecrets {
		key = strings.TrimSpace(key)
		if !aiprovidercreds.IsKnownSecretKey(key) {
			continue
		}
		if err := aiprovidercreds.ClearSecretKeyed(ctx, pool, scope, orgID, provider, key); err != nil {
			return err
		}
	}
	for key, val := range secrets {
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if !aiprovidercreds.IsKnownSecretKey(key) || val == "" || val == placeholderSecretResponse {
			continue
		}
		if err := aiprovidercreds.StoreSecretKeyed(ctx, pool, scope, orgID, provider, key, secretsKey, val); err != nil {
			return err
		}
	}
	return nil
}

func credentialPublicJSON(c aiprovidercreds.Credential, apiKeyConfigured bool) map[string]any {
	secretsConfigured := map[string]bool{}
	for k, v := range c.SecretsConfigured {
		if v {
			secretsConfigured[k] = true
		}
	}
	if apiKeyConfigured {
		secretsConfigured[aiprovidercreds.SecretKeyAPIKey] = true
	}
	authMode := aiprovider.AuthModeFromSettings(aiprovider.ProviderName(c.Provider), c.Settings)
	out := map[string]any{
		"provider":                      c.Provider,
		"enabled":                       c.Enabled,
		"apiKeyConfigured":              apiKeyConfigured,
		"apiKey":                        maskSecret(ternarySecret(apiKeyConfigured)),
		"secretsConfigured":             secretsConfigured,
		"authMode":                      authMode,
		"settings":                      c.Settings,
		"updatedAt":                     c.UpdatedAt.UTC().Format(time.RFC3339),
		"awsAccessKeyIdConfigured":      secretsConfigured[aiprovidercreds.SecretKeyAWSAccessKeyID],
		"awsSecretAccessKeyConfigured":  secretsConfigured[aiprovidercreds.SecretKeyAWSSecretAccessKey],
		"serviceAccountJsonConfigured":  secretsConfigured[aiprovidercreds.SecretKeyServiceAccountJSON],
	}
	if c.UpdatedBy != nil {
		out["updatedBy"] = c.UpdatedBy.String()
	}
	return out
}

func writeAIProviderTestError(w http.ResponseWriter, err error) {
	errType := aiprovider.ClassifyError(err)
	msg := "Provider test failed: " + err.Error()
	switch errType {
	case aiprovider.ErrorTypeConfig:
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
	case aiprovider.ErrorTypeAuth:
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, msg)
	case aiprovider.ErrorTypeQuota:
		apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, msg)
	default:
		apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, msg)
	}
}

// collectSecretsFromBody builds secret_key→value and clear lists from optional multi-secret fields.
func collectSecretsFromBody(
	apiKey *string,
	clearAPIKey bool,
	awsAccessKeyID, awsSecretAccessKey, serviceAccountJSON *string,
	clearAWSAccessKeyID, clearAWSSecretAccessKey, clearServiceAccountJSON bool,
) (secrets map[string]string, clear []string, needsWrite bool) {
	secrets = map[string]string{}
	if clearAPIKey {
		clear = append(clear, aiprovidercreds.SecretKeyAPIKey)
	} else if apiKey != nil {
		k := strings.TrimSpace(*apiKey)
		if k != "" && k != placeholderSecretResponse {
			secrets[aiprovidercreds.SecretKeyAPIKey] = k
			needsWrite = true
		}
	}
	if clearAWSAccessKeyID {
		clear = append(clear, aiprovidercreds.SecretKeyAWSAccessKeyID)
	} else if awsAccessKeyID != nil {
		k := strings.TrimSpace(*awsAccessKeyID)
		if k != "" && k != placeholderSecretResponse {
			secrets[aiprovidercreds.SecretKeyAWSAccessKeyID] = k
			needsWrite = true
		}
	}
	if clearAWSSecretAccessKey {
		clear = append(clear, aiprovidercreds.SecretKeyAWSSecretAccessKey)
	} else if awsSecretAccessKey != nil {
		k := strings.TrimSpace(*awsSecretAccessKey)
		if k != "" && k != placeholderSecretResponse {
			secrets[aiprovidercreds.SecretKeyAWSSecretAccessKey] = k
			needsWrite = true
		}
	}
	if clearServiceAccountJSON {
		clear = append(clear, aiprovidercreds.SecretKeyServiceAccountJSON)
	} else if serviceAccountJSON != nil {
		k := strings.TrimSpace(*serviceAccountJSON)
		if k != "" && k != placeholderSecretResponse {
			secrets[aiprovidercreds.SecretKeyServiceAccountJSON] = k
			needsWrite = true
		}
	}
	if len(clear) > 0 {
		needsWrite = true
	}
	return secrets, clear, needsWrite
}
