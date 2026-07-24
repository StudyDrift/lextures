package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
	"github.com/lextures/lextures/server/internal/repos/aiprovidercreds"
	tenantaisettings "github.com/lextures/lextures/server/internal/repos/tenantaisettings"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

func (d Deps) aiProviderAbstractionEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().AiProviderAbstractionEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "AI provider abstraction is not enabled.")
		return false
	}
	return true
}

func (d Deps) registerAIProviderSettingsRoutes(r chi.Router) {
	r.Get("/api/v1/admin/ai-settings", d.handleGetAdminAISettings())
	r.Put("/api/v1/admin/ai-settings", d.handlePutAdminAISettings())
	r.Post("/api/v1/admin/ai-settings/test", d.handlePostAdminAISettingsTest())
	d.registerAIProviderCredentialRoutes(r)
}

func (d Deps) handleGetAdminAISettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiProviderAbstractionEnabled(w) {
			return
		}
		_, orgID, ok := d.requireOrgAdminForAIConfig(w, r)
		if !ok {
			return
		}
		row, err := tenantaisettings.GetByOrgID(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load AI provider settings.")
			return
		}
		creds, _ := aiprovidercreds.ListByScope(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID)
		byokConfigured := false
		credSummaries := make([]map[string]any, 0, len(creds))
		for _, c := range creds {
			if c.SecretConfigured {
				byokConfigured = true
			}
			aiprovider.RecordCredentialConfigured(aiprovidercreds.ScopeOrg, c.Provider, c.SecretConfigured)
			credSummaries = append(credSummaries, credentialPublicJSON(c, c.SecretConfigured))
		}
		if !byokConfigured {
			if bc, err := tenantaisettings.BYOKConfigured(r.Context(), d.Pool, orgID); err == nil {
				byokConfigured = bc
			}
		}
		resp := map[string]any{
			"orgId":             orgID.String(),
			"provider":          string(aiprovider.ProviderOpenRouter),
			"modelAlias":        string(aiprovider.AliasClaude35Sonnet),
			"fallbackProvider":  nil,
			"byokConfigured":    byokConfigured,
			"settings":          map[string]any{},
			"credentials":       credSummaries,
			"providers":         providerNamesJSON(),
			"modelAliases":      aiprovider.ListModelAliases(),
			"modelAliasCatalog": aiprovider.ListModelAliasInfos(),
			"registryVersion":   aiprovider.RegistryVersion,
		}
		if row != nil {
			resp["provider"] = row.Provider
			resp["modelAlias"] = row.ModelAlias
			if row.FallbackProvider != nil {
				resp["fallbackProvider"] = *row.FallbackProvider
			}
			resp["settings"] = row.Settings
			resp["updatedAt"] = row.UpdatedAt.UTC().Format(time.RFC3339)
			if row.UpdatedBy != nil {
				resp["updatedBy"] = row.UpdatedBy.String()
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (d Deps) handlePutAdminAISettings() http.HandlerFunc {
	type providerCredBody struct {
		Provider                  string         `json:"provider"`
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
	type body struct {
		Provider         string             `json:"provider"`
		ModelAlias       string             `json:"modelAlias"`
		FallbackProvider *string            `json:"fallbackProvider"`
		BYOKAPIKey       *string            `json:"byokApiKey"`
		ClearBYOKAPIKey  bool               `json:"clearByokApiKey"`
		Settings         map[string]any     `json:"settings"`
		Credentials      []providerCredBody `json:"credentials"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiProviderAbstractionEnabled(w) {
			return
		}
		actorID, orgID, ok := d.requireOrgAdminForAIConfig(w, r)
		if !ok {
			return
		}
		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		provider := strings.TrimSpace(b.Provider)
		if provider == "" {
			provider = string(aiprovider.ProviderOpenRouter)
		}
		modelAlias := strings.TrimSpace(b.ModelAlias)
		if modelAlias == "" {
			modelAlias = string(aiprovider.AliasClaude35Sonnet)
		}
		if _, err := aiprovider.ResolveModelID(modelAlias, aiprovider.ProviderName(provider)); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid provider or model alias.")
			return
		}
		policy, _ := aiprovidercreds.GetTenantBYOKPolicy(r.Context(), d.Pool)
		if !policy.AllowTenantProvider(provider) {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Platform policy does not allow this AI provider for tenant BYOK.")
			return
		}
		var fallback *string
		if b.FallbackProvider != nil {
			fp := strings.TrimSpace(*b.FallbackProvider)
			if fp != "" {
				fallback = &fp
			}
		}

		needsSecretWrite := b.BYOKAPIKey != nil &&
			strings.TrimSpace(*b.BYOKAPIKey) != "" &&
			strings.TrimSpace(*b.BYOKAPIKey) != placeholderSecretResponse
		for _, c := range b.Credentials {
			_, _, need := collectSecretsFromBody(
				c.APIKey, c.ClearAPIKey,
				c.AWSAccessKeyID, c.AWSSecretAccessKey, c.ServiceAccountJSON,
				c.ClearAWSAccessKeyID, c.ClearAWSSecretAccessKey, c.ClearServiceAccountJSON,
			)
			if need {
				needsSecretWrite = true
				break
			}
		}
		secretsKey := d.effectiveConfig().PlatformSecretsKey
		if needsSecretWrite && len(secretsKey) != 32 {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal,
				"Set PLATFORM_SECRETS_KEY to a base64-encoded 32-byte key (e.g. openssl rand -base64 32) before storing BYOK credentials.")
			return
		}

		byokRef := ""
		if b.ClearBYOKAPIKey {
			_ = tenantaisettings.ClearBYOK(r.Context(), d.Pool, orgID)
			_ = aiprovidercreds.ClearSecret(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID, provider)
		} else if b.BYOKAPIKey != nil {
			key := strings.TrimSpace(*b.BYOKAPIKey)
			if key != "" && key != placeholderSecretResponse {
				if err := tenantaisettings.StoreBYOK(r.Context(), d.Pool, orgID, secretsKey, key); err != nil {
					if err == appsecrets.ErrInvalidKey {
						apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal,
							"Set PLATFORM_SECRETS_KEY to a base64-encoded 32-byte key before storing BYOK credentials.")
						return
					}
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not store BYOK key.")
					return
				}
				// Dual-write into multi-provider store.
				_ = aiprovidercreds.Upsert(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID, provider, aiprovidercreds.UpsertInput{
					UpdatedBy: &actorID,
				})
				if err := aiprovidercreds.StoreSecret(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID, provider, secretsKey, key); err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not store BYOK key.")
					return
				}
				byokRef = tenantaisettings.DefaultBYOKRef()
			}
		}

		for _, c := range b.Credentials {
			p := strings.TrimSpace(c.Provider)
			if p == "" || !isKnownAIProvider(p) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credentials[].provider.")
				return
			}
			if !policy.AllowTenantProvider(p) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Platform policy does not allow provider "+p+" for tenant BYOK.")
				return
			}
			if c.ClearAPIKey && c.APIKey != nil {
				s := strings.TrimSpace(*c.APIKey)
				if s != "" && s != placeholderSecretResponse {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Cannot set apiKey and clearApiKey together.")
					return
				}
			}
			if err := aiprovidercreds.Upsert(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID, p, aiprovidercreds.UpsertInput{
				Enabled:     c.Enabled,
				Settings:    c.Settings,
				SetSettings: c.Settings != nil,
				UpdatedBy:   &actorID,
			}); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save provider credential.")
				return
			}
			secrets, clearKeys, needSecrets := collectSecretsFromBody(
				c.APIKey, c.ClearAPIKey,
				c.AWSAccessKeyID, c.AWSSecretAccessKey, c.ServiceAccountJSON,
				c.ClearAWSAccessKeyID, c.ClearAWSSecretAccessKey, c.ClearServiceAccountJSON,
			)
			if needSecrets {
				if err := applyProviderSecrets(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID, p, secretsKey, secrets, clearKeys); err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not store provider secrets.")
					return
				}
				// Dual-write primary provider api_key into legacy BYOK for transition.
				if p == provider {
					if k, ok := secrets[aiprovidercreds.SecretKeyAPIKey]; ok {
						_ = tenantaisettings.StoreBYOK(r.Context(), d.Pool, orgID, secretsKey, k)
						byokRef = tenantaisettings.DefaultBYOKRef()
					}
					for _, ck := range clearKeys {
						if ck == aiprovidercreds.SecretKeyAPIKey {
							_ = tenantaisettings.ClearBYOK(r.Context(), d.Pool, orgID)
						}
					}
				}
			}
			configured, _ := aiprovidercreds.SecretConfigured(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID, p)
			aiprovider.RecordCredentialConfigured(aiprovidercreds.ScopeOrg, p, configured)
		}

		before, _ := tenantaisettings.GetByOrgID(r.Context(), d.Pool, orgID)
		if err := tenantaisettings.Upsert(r.Context(), d.Pool, orgID, tenantaisettings.UpsertInput{
			Provider:         provider,
			ModelAlias:       modelAlias,
			FallbackProvider: fallback,
			BYOKSecretRef:    byokRef,
			Settings:         b.Settings,
			UpdatedBy:        actorID,
		}); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save AI provider settings.")
			return
		}
		if d.aiProviderResolver() != nil {
			d.aiProviderResolver().InvalidateCache(orgID)
		}
		// Audit without secret values (AP.2 AC-6 / FR-8).
		auditAfter := map[string]any{
			"provider":         provider,
			"modelAlias":       modelAlias,
			"fallbackProvider": fallback,
			"settings":         b.Settings,
			"clearByokApiKey":  b.ClearBYOKAPIKey,
			"byokApiKeySet":    b.BYOKAPIKey != nil && strings.TrimSpace(derefStr(b.BYOKAPIKey)) != "" && strings.TrimSpace(derefStr(b.BYOKAPIKey)) != placeholderSecretResponse,
			"credentialsCount": len(b.Credentials),
		}
		afterJSON, _ := json.Marshal(auditAfter)
		beforeJSON, _ := json.Marshal(redactTenantAISettings(before))
		orgPtr := &orgID
		_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
			OrgID:       orgPtr,
			EventType:   auditservice.EventAIConfigChange,
			ActorID:     actorID,
			TargetType:  aiProviderSettingsTargetType(),
			TargetID:    &orgID,
			BeforeValue: beforeJSON,
			AfterValue:  afterJSON,
		})
		byokConfigured, _ := tenantaisettings.BYOKConfigured(r.Context(), d.Pool, orgID)
		if !byokConfigured {
			if n, err := aiprovidercreds.ListByScope(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID); err == nil {
				for _, c := range n {
					if c.SecretConfigured {
						byokConfigured = true
						break
					}
				}
			}
		}
		creds, _ := aiprovidercreds.ListByScope(r.Context(), d.Pool, aiprovidercreds.ScopeOrg, &orgID)
		credSummaries := make([]map[string]any, 0, len(creds))
		for _, c := range creds {
			credSummaries = append(credSummaries, credentialPublicJSON(c, c.SecretConfigured))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"orgId":            orgID.String(),
			"provider":         provider,
			"modelAlias":       modelAlias,
			"fallbackProvider": fallback,
			"byokConfigured":   byokConfigured,
			"settings":         b.Settings,
			"credentials":      credSummaries,
		})
	}
}

func (d Deps) handlePostAdminAISettingsTest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.aiProviderAbstractionEnabled(w) {
			return
		}
		_, orgID, ok := d.requireOrgAdminForAIConfig(w, r)
		if !ok {
			return
		}
		resolver := d.aiProviderResolver()
		if resolver == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, "AI provider resolver is not configured.")
			return
		}
		start := time.Now()
		got, meta, err := resolver.Complete(r.Context(), &orgID, "", []aiprovider.Message{
			{Role: "user", Content: "Hello"},
		})
		if err != nil {
			writeAIProviderTestError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":               true,
			"provider":         meta.Provider,
			"authMode":         meta.AuthMode,
			"modelAlias":       meta.ModelAlias,
			"modelId":          meta.ModelID,
			"latencyMs":        meta.Latency.Milliseconds(),
			"totalLatencyMs":   time.Since(start).Milliseconds(),
			"promptTokens":     got.Usage.PromptTokens,
			"completionTokens": got.Usage.CompletionTokens,
			"responsePreview":  truncatePreview(got.Text, 200),
		})
	}
}

func providerNamesJSON() []string {
	names := aiprovider.ListProviders()
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = string(n)
	}
	return out
}

func truncatePreview(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func aiProviderSettingsTargetType() *string {
	s := "tenant_ai_settings"
	return &s
}

func redactTenantAISettings(row *tenantaisettings.Row) map[string]any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"provider":         row.Provider,
		"modelAlias":       row.ModelAlias,
		"fallbackProvider": row.FallbackProvider,
		"byokConfigured":   row.BYOKSecretRef != "",
		"settings":         row.Settings,
	}
}

