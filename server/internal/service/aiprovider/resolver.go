package aiprovider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	tenantaisettings "github.com/lextures/lextures/server/internal/repos/tenantaisettings"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

const settingsCacheTTL = 5 * time.Minute

// ResolverConfig controls tenant provider resolution.
type ResolverConfig struct {
	AbstractionEnabled bool
	PlatformAPIKey     string
	SecretsKey         []byte
	DryRun             bool
}

type cachedSettings struct {
	settings  *tenantaisettings.Row
	expiresAt time.Time
}

// Resolver selects the effective AI provider per tenant (FR-3, FR-4, FR-7).
type Resolver struct {
	pool     *pgxpool.Pool
	factory  Factory
	cfg      ResolverConfig
	cacheMu  sync.RWMutex
	cache    map[uuid.UUID]cachedSettings
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

// Complete runs a chat completion with tenant-aware provider selection and optional fallback.
func (r *Resolver) Complete(
	ctx context.Context,
	orgID *uuid.UUID,
	modelOverride string,
	messages []Message,
	opts ...ChatOptions,
) (ChatResult, CallMeta, error) {
	if r == nil {
		return ChatResult{}, CallMeta{}, fmt.Errorf("aiprovider: nil resolver")
	}
	if r.cfg.DryRun {
		p := &DryRunProvider{}
		start := time.Now()
		got, err := p.Complete(ctx, "dry-run", messages, opts...)
		meta := CallMeta{Provider: ProviderDryRun, ModelAlias: "dry-run", ModelID: "dry-run", Latency: time.Since(start)}
		if err == nil {
			recordLatency(meta.Provider, meta.ModelAlias, meta.Latency.Seconds())
			recordCostUSD(meta.Provider, got.Usage.CostUSD)
		}
		return got, meta, err
	}

	settings, apiKey, err := r.resolveTenant(ctx, orgID)
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
		modelID = modelOverride
	}

	primary, err := r.factory.Build(providerName, apiKey, settings.Extra)
	if err != nil {
		return ChatResult{}, CallMeta{}, err
	}
	got, meta, err := r.callProvider(ctx, primary, providerName, modelAlias, modelID, messages, opts...)
	if err == nil || settings.FallbackProvider == nil || !IsRetryable(err) {
		return got, meta, err
	}
	fallbackName := *settings.FallbackProvider
	fallbackKey, ferr := r.apiKeyForProvider(ctx, orgID, fallbackName, settings)
	if ferr != nil {
		fallbackKey = r.cfg.PlatformAPIKey
	}
	fallbackModelID, ferr := ResolveModelID(modelAlias, fallbackName)
	if ferr != nil {
		return got, meta, err
	}
	fallback, ferr := r.factory.Build(fallbackName, fallbackKey, settings.Extra)
	if ferr != nil {
		return got, meta, err
	}
	got2, meta2, err2 := r.callProvider(ctx, fallback, fallbackName, modelAlias, fallbackModelID, messages, opts...)
	if err2 != nil {
		return got, meta, err
	}
	return got2, meta2, nil
}

func (r *Resolver) callProvider(
	ctx context.Context,
	p Provider,
	providerName ProviderName,
	modelAlias, modelID string,
	messages []Message,
	opts ...ChatOptions,
) (ChatResult, CallMeta, error) {
	start := time.Now()
	got, err := p.Complete(ctx, modelID, messages, opts...)
	latency := time.Since(start)
	meta := CallMeta{
		Provider:   providerName,
		ModelAlias: modelAlias,
		ModelID:    modelID,
		Latency:    latency,
	}
	if err != nil {
		recordError(providerName, "complete")
		return ChatResult{}, meta, err
	}
	meta.Usage = got.Usage
	recordLatency(providerName, modelAlias, latency.Seconds())
	recordCostUSD(providerName, got.Usage.CostUSD)
	return got, meta, nil
}

func (r *Resolver) resolveTenant(ctx context.Context, orgID *uuid.UUID) (Settings, string, error) {
	defaults := Settings{
		Provider:   ProviderOpenRouter,
		ModelAlias: string(AliasClaude35Sonnet),
		Extra:      nil,
	}
	apiKey := r.cfg.PlatformAPIKey
	if !r.cfg.AbstractionEnabled || orgID == nil {
		return defaults, apiKey, nil
	}
	row, err := r.loadSettings(ctx, *orgID)
	if err != nil {
		return Settings{}, "", err
	}
	if row == nil {
		return defaults, apiKey, nil
	}
	settings := Settings{
		Provider:       ProviderName(row.Provider),
		ModelAlias:     row.ModelAlias,
		BYOKConfigured: row.BYOKSecretRef != "",
		Extra:          row.Settings,
	}
	if row.FallbackProvider != nil && *row.FallbackProvider != "" {
		fp := ProviderName(*row.FallbackProvider)
		settings.FallbackProvider = &fp
	}
	key, err := r.apiKeyForProvider(ctx, orgID, settings.Provider, settings)
	if err != nil {
		return Settings{}, "", err
	}
	if key != "" {
		apiKey = key
	}
	return settings, apiKey, nil
}

func (r *Resolver) apiKeyForProvider(ctx context.Context, orgID *uuid.UUID, provider ProviderName, settings Settings) (string, error) {
	if orgID == nil || r.pool == nil {
		return r.cfg.PlatformAPIKey, nil
	}
	if settings.BYOKConfigured {
		key, err := tenantaisettings.DecryptBYOK(ctx, r.pool, *orgID, r.cfg.SecretsKey)
		if err == nil && key != "" {
			return key, nil
		}
	}
	if provider == ProviderOpenRouter {
		return r.cfg.PlatformAPIKey, nil
	}
	return "", fmt.Errorf("aiprovider: no API key for provider %s", provider)
}

// GetSettings returns the effective tenant AI settings for admin display.
func (r *Resolver) GetSettings(ctx context.Context, orgID uuid.UUID) (Settings, error) {
	settings, _, err := r.resolveTenant(ctx, &orgID)
	return settings, err
}