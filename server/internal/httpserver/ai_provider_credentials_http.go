package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
	"github.com/lextures/lextures/server/internal/repos/aiprovidercreds"
	"github.com/lextures/lextures/server/internal/repos/platformconfig"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

func (d Deps) registerAIProviderCredentialRoutes(r chi.Router) {
	r.Get("/api/v1/settings/ai/providers", d.handleListPlatformAIProviders())
	r.Put("/api/v1/settings/ai/providers", d.handlePutPlatformAIProviderPolicy())
	r.Put("/api/v1/settings/ai/providers/{provider}", d.handlePutPlatformAIProvider())
	r.Delete("/api/v1/settings/ai/providers/{provider}", d.handleDeletePlatformAIProvider())
}

func (d Deps) handleListPlatformAIProviders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiProviderAbstractionEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		creds, err := aiprovidercreds.ListByScope(r.Context(), d.Pool, aiprovidercreds.ScopePlatform, nil)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load AI provider credentials.")
			return
		}
		// Dual-read: treat legacy OpenRouter column as configured when secret row is missing.
		cfg := d.effectiveConfig()
		legacyOR := strings.TrimSpace(cfg.OpenRouterAPIKey) != ""
		out := make([]map[string]any, 0, len(aiprovider.ListProviders()))
		seen := map[string]bool{}
		for _, c := range creds {
			configured := c.SecretConfigured
			if c.Provider == string(aiprovider.ProviderOpenRouter) && legacyOR {
				configured = true
			}
			aiprovider.RecordCredentialConfigured(aiprovidercreds.ScopePlatform, c.Provider, configured)
			out = append(out, credentialPublicJSON(c, configured))
			seen[c.Provider] = true
		}
		for _, name := range aiprovider.ListProviders() {
			p := string(name)
			if seen[p] {
				continue
			}
			configured := p == string(aiprovider.ProviderOpenRouter) && legacyOR
			aiprovider.RecordCredentialConfigured(aiprovidercreds.ScopePlatform, p, configured)
			out = append(out, map[string]any{
				"provider":                     p,
				"enabled":                      true,
				"apiKeyConfigured":             configured,
				"apiKey":                       maskSecret(ternarySecret(configured)),
				"secretsConfigured":            map[string]bool{},
				"authMode":                     aiprovider.AuthModeAPIKey,
				"settings":                     map[string]any{},
				"awsAccessKeyIdConfigured":     false,
				"awsSecretAccessKeyConfigured": false,
				"serviceAccountJsonConfigured": false,
			})
		}
		policy, _ := aiprovidercreds.GetTenantBYOKPolicy(r.Context(), d.Pool)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"credentials":             out,
			"providers":               providerNamesJSON(),
			"tenantByokAllowed":       policy.Allowed,
			"tenantAllowedProviders":  policy.AllowedProviders,
		})
	}
}

func (d Deps) handlePutPlatformAIProviderPolicy() http.HandlerFunc {
	type body struct {
		TenantByokAllowed      *bool     `json:"tenantByokAllowed"`
		TenantAllowedProviders *[]string `json:"tenantAllowedProviders"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiProviderAbstractionEnabled(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if b.TenantByokAllowed == nil && b.TenantAllowedProviders == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide tenantByokAllowed and/or tenantAllowedProviders.")
			return
		}
		before, _ := aiprovidercreds.GetTenantBYOKPolicy(r.Context(), d.Pool)
		if err := aiprovidercreds.SetTenantBYOKPolicy(r.Context(), d.Pool, b.TenantByokAllowed, b.TenantAllowedProviders); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save BYOK policy.")
			return
		}
		after, _ := aiprovidercreds.GetTenantBYOKPolicy(r.Context(), d.Pool)
		beforeJSON, _ := json.Marshal(map[string]any{
			"tenantByokAllowed": before.Allowed, "tenantAllowedProviders": before.AllowedProviders,
		})
		afterJSON, _ := json.Marshal(map[string]any{
			"tenantByokAllowed": after.Allowed, "tenantAllowedProviders": after.AllowedProviders,
		})
		tt := "ai_provider_policy"
		_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
			EventType:   auditservice.EventAIConfigChange,
			ActorID:     actorID,
			TargetType:  &tt,
			BeforeValue: beforeJSON,
			AfterValue:  afterJSON,
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tenantByokAllowed":      after.Allowed,
			"tenantAllowedProviders": after.AllowedProviders,
		})
	}
}

func (d Deps) handlePutPlatformAIProvider() http.HandlerFunc {
	type body struct {
		Enabled                   *bool          `json:"enabled"`
		APIKey                    *string        `json:"apiKey"`
		ClearAPIKey               bool           `json:"clearApiKey"`
		AWSAccessKeyID            *string        `json:"awsAccessKeyId"`
		ClearAWSAccessKeyID       bool           `json:"clearAwsAccessKeyId"`
		AWSSecretAccessKey        *string        `json:"awsSecretAccessKey"`
		ClearAWSSecretAccessKey   bool           `json:"clearAwsSecretAccessKey"`
		ServiceAccountJSON        *string        `json:"serviceAccountJson"`
		ClearServiceAccountJSON   bool           `json:"clearServiceAccountJson"`
		Settings                  map[string]any `json:"settings"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiProviderAbstractionEnabled(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		provider := strings.TrimSpace(chi.URLParam(r, "provider"))
		if !isKnownAIProvider(provider) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown AI provider.")
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if b.ClearAPIKey && b.APIKey != nil {
			s := strings.TrimSpace(*b.APIKey)
			if s != "" && s != placeholderSecretResponse {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Cannot set apiKey and clearApiKey together.")
				return
			}
		}
		secrets, clearKeys, needsSecretWrite := collectSecretsFromBody(
			b.APIKey, b.ClearAPIKey,
			b.AWSAccessKeyID, b.AWSSecretAccessKey, b.ServiceAccountJSON,
			b.ClearAWSAccessKeyID, b.ClearAWSSecretAccessKey, b.ClearServiceAccountJSON,
		)
		before, _ := aiprovidercreds.Get(r.Context(), d.Pool, aiprovidercreds.ScopePlatform, nil, provider)
		if err := aiprovidercreds.Upsert(r.Context(), d.Pool, aiprovidercreds.ScopePlatform, nil, provider, aiprovidercreds.UpsertInput{
			Enabled:     b.Enabled,
			Settings:    b.Settings,
			SetSettings: b.Settings != nil,
			UpdatedBy:   &actorID,
		}); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save provider credential.")
			return
		}
		secretsKey := d.effectiveConfig().PlatformSecretsKey
		hasPlaintext := len(secrets) > 0
		if hasPlaintext && len(secretsKey) != 32 {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal,
				"Set PLATFORM_SECRETS_KEY to a base64-encoded 32-byte key (e.g. openssl rand -base64 32) before storing provider credentials.")
			return
		}
		if needsSecretWrite {
			if err := applyProviderSecrets(r.Context(), d.Pool, aiprovidercreds.ScopePlatform, nil, provider, secretsKey, secrets, clearKeys); err != nil {
				if err == appsecrets.ErrInvalidKey {
					apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal,
						"Set PLATFORM_SECRETS_KEY to a base64-encoded 32-byte key before storing provider credentials.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not store provider secrets.")
				return
			}
			// Dual-write OpenRouter into legacy column for transition.
			if provider == string(aiprovider.ProviderOpenRouter) {
				if k, ok := secrets[aiprovidercreds.SecretKeyAPIKey]; ok {
					if _, err := platformconfig.Upsert(r.Context(), d.Pool, &platformconfig.Write{OpenRouterAPIKey: &k}); err != nil {
						apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not dual-write OpenRouter key.")
						return
					}
				}
				for _, ck := range clearKeys {
					if ck == aiprovidercreds.SecretKeyAPIKey {
						_ = platformconfig.ClearOpenRouterAPIKey(r.Context(), d.Pool)
					}
				}
			}
		}
		d.reloadPlatformAIClients()
		if res := d.aiProviderResolver(); res != nil {
			res.InvalidateAllCaches()
		}
		after, _ := aiprovidercreds.Get(r.Context(), d.Pool, aiprovidercreds.ScopePlatform, nil, provider)
		configured := after != nil && after.SecretConfigured
		if provider == string(aiprovider.ProviderOpenRouter) && strings.TrimSpace(d.effectiveConfig().OpenRouterAPIKey) != "" {
			configured = true
		}
		aiprovider.RecordCredentialConfigured(aiprovidercreds.ScopePlatform, provider, configured)
		d.auditAICredentialChange(r, actorID, nil, "platform_ai_provider", provider, before, after, b.ClearAPIKey, b.APIKey != nil || len(secrets) > 0)
		if after == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Credential missing after save.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(credentialPublicJSON(*after, configured))
	}
}

func (d Deps) handleDeletePlatformAIProvider() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiProviderAbstractionEnabled(w) {
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		provider := strings.TrimSpace(chi.URLParam(r, "provider"))
		if !isKnownAIProvider(provider) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown AI provider.")
			return
		}
		before, _ := aiprovidercreds.Get(r.Context(), d.Pool, aiprovidercreds.ScopePlatform, nil, provider)
		if err := aiprovidercreds.Delete(r.Context(), d.Pool, aiprovidercreds.ScopePlatform, nil, provider); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not clear provider credential.")
			return
		}
		if provider == string(aiprovider.ProviderOpenRouter) {
			_ = platformconfig.ClearOpenRouterAPIKey(r.Context(), d.Pool)
		}
		d.reloadPlatformAIClients()
		if res := d.aiProviderResolver(); res != nil {
			res.InvalidateAllCaches()
		}
		aiprovider.RecordCredentialConfigured(aiprovidercreds.ScopePlatform, provider, false)
		d.auditAICredentialChange(r, actorID, nil, "platform_ai_provider", provider, before, nil, true, false)
		w.WriteHeader(http.StatusNoContent)
	}
}

func platformCredentialJSON(c aiprovidercreds.Credential, configured bool) map[string]any {
	return credentialPublicJSON(c, configured)
}

func ternarySecret(configured bool) string {
	if configured {
		return "x"
	}
	return ""
}

func isKnownAIProvider(name string) bool {
	for _, p := range aiprovider.ListProviders() {
		if string(p) == name {
			return true
		}
	}
	return false
}

func (d Deps) reloadPlatformAIClients() {
	if d.Pool == nil || d.Platform == nil {
		return
	}
	dbRow, err := platformconfig.Get(context.Background(), d.Pool)
	if err != nil {
		return
	}
	merged := platformconfig.Merge(d.Config, dbRow)
	if err := merged.Validate(); err != nil {
		return
	}
	d.Platform.Reload(merged)
}

func (d Deps) auditAICredentialChange(
	r *http.Request,
	actorID uuid.UUID,
	orgID *uuid.UUID,
	targetType, provider string,
	before, after *aiprovidercreds.Credential,
	cleared, keyTouched bool,
) {
	redacted := func(c *aiprovidercreds.Credential) map[string]any {
		if c == nil {
			return nil
		}
		return map[string]any{
			"provider":         c.Provider,
			"enabled":          c.Enabled,
			"apiKeyConfigured": c.SecretConfigured,
			"settings":         c.Settings,
		}
	}
	afterMap := redacted(after)
	if afterMap != nil {
		if cleared {
			afterMap["apiKeyConfigured"] = false
		} else if keyTouched {
			afterMap["apiKeyConfigured"] = true
		}
	}
	beforeJSON, _ := json.Marshal(redacted(before))
	afterJSON, _ := json.Marshal(map[string]any{
		"provider": provider,
		"change":   afterMap,
		"cleared":  cleared,
	})
	tt := targetType
	_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
		OrgID:       orgID,
		EventType:   auditservice.EventAIConfigChange,
		ActorID:     actorID,
		TargetType:  &tt,
		TargetID:    orgID,
		BeforeValue: beforeJSON,
		AfterValue:  afterJSON,
	})
}
