package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
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
		byokConfigured := false
		if bc, err := tenantaisettings.BYOKConfigured(r.Context(), d.Pool, orgID); err == nil {
			byokConfigured = bc
		}
		resp := map[string]any{
			"orgId":            orgID.String(),
			"provider":         string(aiprovider.ProviderOpenRouter),
			"modelAlias":       string(aiprovider.AliasClaude35Sonnet),
			"fallbackProvider": nil,
			"byokConfigured":   byokConfigured,
			"settings":         map[string]any{},
			"providers":        providerNamesJSON(),
			"modelAliases":     aiprovider.ListModelAliases(),
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
	type body struct {
		Provider         string         `json:"provider"`
		ModelAlias       string         `json:"modelAlias"`
		FallbackProvider *string        `json:"fallbackProvider"`
		BYOKAPIKey       *string        `json:"byokApiKey"`
		Settings         map[string]any `json:"settings"`
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
		var fallback *string
		if b.FallbackProvider != nil {
			fp := strings.TrimSpace(*b.FallbackProvider)
			if fp != "" {
				fallback = &fp
			}
		}
		byokRef := ""
		if b.BYOKAPIKey != nil {
			key := strings.TrimSpace(*b.BYOKAPIKey)
			if key != "" && key != placeholderSecretResponse {
				secretsKey := d.effectiveConfig().PlatformSecretsKey
				if len(secretsKey) != 32 {
					apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Platform secrets key is not configured for BYOK storage.")
					return
				}
				if err := tenantaisettings.StoreBYOK(r.Context(), d.Pool, orgID, secretsKey, key); err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not store BYOK key.")
					return
				}
				byokRef = tenantaisettings.DefaultBYOKRef()
			}
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
		afterJSON, _ := json.Marshal(b)
		beforeJSON, _ := json.Marshal(before)
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"orgId":            orgID.String(),
			"provider":         provider,
			"modelAlias":       modelAlias,
			"fallbackProvider": fallback,
			"byokConfigured":   byokConfigured,
			"settings":         b.Settings,
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
			apierr.WriteJSON(w, http.StatusBadGateway, apierr.CodeInternal, "Provider test failed: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":               true,
			"provider":         meta.Provider,
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

func (d Deps) aiProviderResolver() *aiprovider.Resolver {
	cfg := d.effectiveConfig()
	return aiprovider.NewResolver(d.Pool, d.openRouterClient(), aiprovider.ResolverConfig{
		AbstractionEnabled: cfg.AiProviderAbstractionEnabled,
		PlatformAPIKey:     cfg.OpenRouterAPIKey,
		SecretsKey:         cfg.PlatformSecretsKey,
	})
}