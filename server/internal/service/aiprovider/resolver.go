package aiprovider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/aiprovidercreds"
	tenantaisettings "github.com/lextures/lextures/server/internal/repos/tenantaisettings"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const settingsCacheTTL = 5 * time.Minute

// ResolverConfig controls tenant provider resolution.
type ResolverConfig struct {
	AbstractionEnabled bool
	PlatformAPIKey     string // legacy platform OpenRouter key (dual-read)
	SecretsKey         []byte
	DryRun             bool
}

type cachedSettings struct {
	settings  *tenantaisettings.Row
	expiresAt time.Time
}

// Resolver selects the effective AI provider per tenant (FR-3, FR-4, FR-7; AP.2 FR-5).
type Resolver struct {
	pool    *pgxpool.Pool
	factory Factory
	cfg     ResolverConfig
	cacheMu sync.RWMutex
	cache   map[uuid.UUID]cachedSettings
}

// NewResolver builds a tenant-aware provider resolver.
func NewResolver(pool *pgxpool.Pool, orClient *openrouter.Client, cfg ResolverConfig) *Resolver {
	return &Resolver{
		pool: pool,
		factory: Factory{
			PlatformOpenRouter: orClient,
		},
		cfg:   cfg,
		cache: make(map[uuid.UUID]cachedSettings),
	}
}

// InvalidateCache drops cached settings for an org (after admin update).
func (r *Resolver) InvalidateCache(orgID uuid.UUID) {
	if r == nil {
		return
	}
	r.cacheMu.Lock()
	delete(r.cache, orgID)
	r.cacheMu.Unlock()
}

// InvalidateAllCaches drops all tenant settings caches (after platform credential change).
func (r *Resolver) InvalidateAllCaches() {
	if r == nil {
		return
	}
	r.cacheMu.Lock()
	clear(r.cache)
	r.cacheMu.Unlock()
}

func (r *Resolver) loadSettings(ctx context.Context, orgID uuid.UUID) (*tenantaisettings.Row, error) {
	if r == nil || r.pool == nil {
		return nil, nil
	}
	now := time.Now()
	r.cacheMu.RLock()
	if c, ok := r.cache[orgID]; ok && now.Before(c.expiresAt) {
		r.cacheMu.RUnlock()
		return c.settings, nil
	}
	r.cacheMu.RUnlock()

	row, err := tenantaisettings.GetByOrgID(ctx, r.pool, orgID)
	if err != nil {
		return nil, err
	}
	r.cacheMu.Lock()
	r.cache[orgID] = cachedSettings{settings: row, expiresAt: now.Add(settingsCacheTTL)}
	r.cacheMu.Unlock()
	return row, nil
}

type providerCall func(ctx context.Context, p Provider, modelID string) (ChatResult, error)

// Complete runs a chat completion with tenant-aware provider selection and optional fallback.
func (r *Resolver) Complete(
	ctx context.Context,
	orgID *uuid.UUID,
	modelOverride string,
	messages []Message,
	opts ...ChatOptions,
) (ChatResult, CallMeta, error) {
	return r.dispatch(ctx, orgID, modelOverride, OpComplete, func(ctx context.Context, p Provider, modelID string) (ChatResult, error) {
		return p.Complete(ctx, modelID, messages, opts...)
	})
}

// CompleteStream runs a streaming chat completion with the same tenant/fallback rules as Complete.
// Capability gaps (ErrNotSupported) do not trigger fallback — only retryable transport errors do (AP.1 AC-3).
func (r *Resolver) CompleteStream(
	ctx context.Context,
	orgID *uuid.UUID,
	modelOverride string,
	messages []Message,
	onChunk ChunkHandler,
	opts ...ChatOptions,
) (ChatResult, CallMeta, error) {
	return r.dispatch(ctx, orgID, modelOverride, OpStream, func(ctx context.Context, p Provider, modelID string) (ChatResult, error) {
		return p.CompleteStream(ctx, modelID, messages, onChunk, opts...)
	})
}

// CompleteVision runs a multimodal completion with the same tenant/fallback rules as Complete.
func (r *Resolver) CompleteVision(
	ctx context.Context,
	orgID *uuid.UUID,
	modelOverride string,
	messages []Message,
	opts ...ChatOptions,
) (ChatResult, CallMeta, error) {
	return r.dispatch(ctx, orgID, modelOverride, OpVision, func(ctx context.Context, p Provider, modelID string) (ChatResult, error) {
		return p.CompleteVision(ctx, modelID, messages, opts...)
	})
}

func (r *Resolver) dispatch(
	ctx context.Context,
	orgID *uuid.UUID,
	modelOverride string,
	operation string,
	call providerCall,
) (ChatResult, CallMeta, error) {
	if r == nil {
		return ChatResult{}, CallMeta{}, fmt.Errorf("aiprovider: nil resolver")
	}
	if r.cfg.DryRun {
		p := &DryRunProvider{}
		start := time.Now()
		got, err := call(ctx, p, "dry-run")
		meta := CallMeta{
			Provider:   ProviderDryRun,
			ModelAlias: "dry-run",
			ModelID:    "dry-run",
			Latency:    time.Since(start),
			Operation:  operation,
		}
		if err == nil {
			meta.Usage = got.Usage
			recordLatency(meta.Provider, meta.ModelAlias, operation, meta.Latency.Seconds())
			recordCostUSD(meta.Provider, got.Usage.CostUSD)
			recordTelemetry(meta.Provider, meta.ModelAlias, "ok", meta.Latency.Seconds(), got.Usage.CostUSD)
		} else {
			recordError(meta.Provider, operation)
			recordTelemetry(meta.Provider, meta.ModelAlias, "error", meta.Latency.Seconds(), 0)
		}
		return got, meta, err
	}

	settings, auth, err := r.resolveTenantAuth(ctx, orgID)
	if err != nil {
		return ChatResult{}, CallMeta{}, err
	}
	providerName := settings.Provider
	modelAlias := settings.ModelAlias
	modelID, err := ResolveModelID(modelAlias, providerName)
	if err != nil {
		return ChatResult{}, CallMeta{}, err
	}
	if modelOverride != "" {
		// Dual-read aliases / OpenRouter ids / native ids (AP.3 FR-6/FR-7).
		resolved, rerr := ResolveModelID(modelOverride, providerName)
		if rerr != nil {
			return ChatResult{}, CallMeta{}, rerr
		}
		modelID = resolved
		modelAlias = modelOverride
	}

	authMode := AuthModeFromSettings(providerName, settings.Extra)
	primary, err := r.factory.BuildWithAuth(providerName, auth, settings.Extra)
	if err != nil {
		return ChatResult{}, CallMeta{Provider: providerName, ModelAlias: modelAlias, ModelID: modelID, Operation: operation, AuthMode: authMode}, err
	}
	got, meta, err := r.callProvider(ctx, primary, providerName, modelAlias, modelID, operation, authMode, call)
	if err == nil || settings.FallbackProvider == nil || !IsRetryable(err) {
		return got, meta, err
	}
	fallbackName := *settings.FallbackProvider
	fallbackAuth, ferr := r.authMaterialForProvider(ctx, orgID, fallbackName)
	if ferr != nil {
		return got, meta, err
	}
	fallbackModelID, ferr := ResolveModelID(modelAlias, fallbackName)
	if ferr != nil {
		return got, meta, err
	}
	fallbackMode := AuthModeFromSettings(fallbackName, settings.Extra)
	fallback, ferr := r.factory.BuildWithAuth(fallbackName, fallbackAuth, settings.Extra)
	if ferr != nil {
		return got, meta, err
	}
	got2, meta2, err2 := r.callProvider(ctx, fallback, fallbackName, modelAlias, fallbackModelID, operation, fallbackMode, call)
	if err2 != nil {
		return got, meta, err
	}
	return got2, meta2, nil
}

func (r *Resolver) callProvider(
	ctx context.Context,
	p Provider,
	providerName ProviderName,
	modelAlias, modelID, operation, authMode string,
	call providerCall,
) (ChatResult, CallMeta, error) {
	start := time.Now()
	got, err := call(ctx, p, modelID)
	latency := time.Since(start)
	meta := CallMeta{
		Provider:   providerName,
		ModelAlias: modelAlias,
		ModelID:    modelID,
		Latency:    latency,
		Operation:  operation,
		AuthMode:   authMode,
	}
	if err != nil {
		recordErrorTyped(providerName, operation, ClassifyError(err))
		recordTelemetry(providerName, modelAlias, "error", latency.Seconds(), 0)
		return ChatResult{}, meta, err
	}
	if ApplyCostEstimate(providerName, modelID, &got.Usage) {
		got.Usage.CostEstimated = true
	}
	meta.Usage = got.Usage
	recordLatency(providerName, modelAlias, operation, latency.Seconds())
	outcome := "ok"
	recordCostUSD(providerName, got.Usage.CostUSD)
	recordTelemetry(providerName, modelAlias, outcome, latency.Seconds(), got.Usage.CostUSD)
	return got, meta, nil
}

func (r *Resolver) resolveTenant(ctx context.Context, orgID *uuid.UUID) (Settings, string, error) {
	settings, auth, err := r.resolveTenantAuth(ctx, orgID)
	return settings, auth.APIKey, err
}

func (r *Resolver) resolveTenantAuth(ctx context.Context, orgID *uuid.UUID) (Settings, AuthMaterial, error) {
	defaults := Settings{
		Provider:   ProviderOpenRouter,
		ModelAlias: string(AliasClaude35Sonnet),
		Extra:      nil,
	}
	if !r.cfg.AbstractionEnabled || orgID == nil {
		auth, err := r.authMaterialForProvider(ctx, nil, ProviderOpenRouter)
		if err != nil {
			return Settings{}, AuthMaterial{}, err
		}
		return defaults, auth, nil
	}
	row, err := r.loadSettings(ctx, *orgID)
	if err != nil {
		return Settings{}, AuthMaterial{}, err
	}
	if row == nil {
		auth, err := r.authMaterialForProvider(ctx, orgID, ProviderOpenRouter)
		if err != nil {
			return Settings{}, AuthMaterial{}, err
		}
		return defaults, auth, nil
	}
	settings := Settings{
		Provider:   ProviderName(row.Provider),
		ModelAlias: row.ModelAlias,
		Extra:      row.Settings,
	}
	if row.FallbackProvider != nil && *row.FallbackProvider != "" {
		fp := ProviderName(*row.FallbackProvider)
		settings.FallbackProvider = &fp
	}

	// Merge org credential settings (azure_base_url, etc.) over tenant_ai_settings.settings.
	if cred, err := aiprovidercreds.Get(ctx, r.pool, aiprovidercreds.ScopeOrg, orgID, string(settings.Provider)); err == nil && cred != nil {
		settings.BYOKConfigured = cred.SecretConfigured || len(cred.SecretsConfigured) > 0 || row.BYOKSecretRef != ""
		if len(cred.Settings) > 0 {
			merged := map[string]any{}
			for k, v := range settings.Extra {
				merged[k] = v
			}
			for k, v := range cred.Settings {
				merged[k] = v
			}
			settings.Extra = merged
		}
	} else {
		settings.BYOKConfigured = row.BYOKSecretRef != ""
	}

	auth, err := r.authMaterialForProvider(ctx, orgID, settings.Provider)
	if err != nil {
		return Settings{}, AuthMaterial{}, err
	}
	return settings, auth, nil
}

// apiKeyForProvider implements AP.2 FR-5 (single api_key dual-read). Prefer authMaterialForProvider.
func (r *Resolver) apiKeyForProvider(ctx context.Context, orgID *uuid.UUID, provider ProviderName) (string, error) {
	auth, err := r.authMaterialForProvider(ctx, orgID, provider)
	if err != nil {
		return "", err
	}
	return auth.APIKey, nil
}

// authMaterialForProvider resolves all secrets for a provider (AP.8 FR-6):
// (1) org credential if present and enabled
// (2) else platform credential
// Dual-reads legacy tenant BYOK and platform openrouter_api_key during transition.
func (r *Resolver) authMaterialForProvider(ctx context.Context, orgID *uuid.UUID, provider ProviderName) (AuthMaterial, error) {
	providerStr := string(provider)
	if orgID != nil && r.pool != nil {
		auth, settings, enabled, err := r.loadAuthMaterial(ctx, aiprovidercreds.ScopeOrg, orgID, providerStr, "")
		if err != nil {
			return AuthMaterial{}, err
		}
		if enabled && (authHasMaterial(auth) || authModeNeedsNoSecret(provider, settings)) {
			return auth, nil
		}
		// Dual-read legacy single BYOK when new store has no key for this provider.
		if auth.APIKey == "" {
			if legacy, err := tenantaisettings.DecryptBYOK(ctx, r.pool, *orgID, r.cfg.SecretsKey); err == nil && legacy != "" {
				row, _ := r.loadSettings(ctx, *orgID)
				if row != nil && row.BYOKSecretRef != "" && (row.Provider == providerStr || row.Provider == "") {
					return AuthMaterial{APIKey: legacy}, nil
				}
			}
		}
	}

	if r.pool != nil {
		auth, settings, enabled, err := r.loadAuthMaterial(ctx, aiprovidercreds.ScopePlatform, nil, providerStr, r.cfg.PlatformAPIKey)
		if err != nil {
			return AuthMaterial{}, err
		}
		if enabled && (authHasMaterial(auth) || authModeNeedsNoSecret(provider, settings)) {
			return auth, nil
		}
	} else if provider == ProviderOpenRouter {
		if k := r.cfg.PlatformAPIKey; k != "" {
			return AuthMaterial{APIKey: k}, nil
		}
	}

	return AuthMaterial{}, fmt.Errorf("aiprovider: AI not configured for provider %s", provider)
}

func (r *Resolver) loadAuthMaterial(
	ctx context.Context,
	scope string,
	orgID *uuid.UUID,
	provider string,
	legacyOpenRouter string,
) (AuthMaterial, map[string]any, bool, error) {
	key, settings, enabled, err := aiprovidercreds.ResolveAPIKey(
		ctx, r.pool, scope, orgID, provider, r.cfg.SecretsKey, legacyOpenRouter,
	)
	if err != nil {
		return AuthMaterial{}, nil, false, err
	}
	auth := AuthMaterial{APIKey: key, Secrets: map[string]string{}}
	if key != "" {
		auth.Secrets[aiprovidercreds.SecretKeyAPIKey] = key
	}
	if len(r.cfg.SecretsKey) == 32 {
		all, err := aiprovidercreds.DecryptAllSecrets(ctx, r.pool, scope, orgID, provider, r.cfg.SecretsKey)
		if err != nil {
			return AuthMaterial{}, nil, false, err
		}
		for k, v := range all {
			auth.Secrets[k] = v
			if k == aiprovidercreds.SecretKeyAPIKey && auth.APIKey == "" {
				auth.APIKey = v
			}
		}
	}
	return auth, settings, enabled, nil
}

func authHasMaterial(auth AuthMaterial) bool {
	if auth.APIKey != "" {
		return true
	}
	if auth.Secret(secretKeyAWSAccessKeyID) != "" || auth.Secret(secretKeyAWSSecretAccessKey) != "" {
		return true
	}
	if auth.Secret(secretKeyServiceAccountJSON) != "" {
		return true
	}
	return false
}

func authModeNeedsNoSecret(provider ProviderName, settings map[string]any) bool {
	mode := AuthModeFromSettings(provider, settings)
	return mode == AuthModeIAMRole || mode == AuthModeADC
}

// GetSettings returns the effective tenant AI settings for admin display.
func (r *Resolver) GetSettings(ctx context.Context, orgID uuid.UUID) (Settings, error) {
	settings, _, err := r.resolveTenant(ctx, &orgID)
	return settings, err
}

// IsCapabilityGap reports whether err is a capability-not-supported failure (no fallback).
func IsCapabilityGap(err error) bool {
	return errors.Is(err, ErrNotSupported)
}
