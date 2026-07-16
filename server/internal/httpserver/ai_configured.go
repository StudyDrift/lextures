package httpserver

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/aiprovidercreds"
	"github.com/lextures/lextures/server/internal/repos/organization"
	tenantaisettings "github.com/lextures/lextures/server/internal/repos/tenantaisettings"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

// Provider-agnostic copy for missing AI configuration (AP.4 FR-8).
const aiNotConfiguredMsg = "AI is not configured. Configure AI under Settings → Intelligence."

// aiProviderResolver returns a tenant-aware provider resolver (AP.1–AP.3).
// Prefer this for all product AI invocations; do not call openRouterClient for inference.
func (d Deps) aiProviderResolver() *aiprovider.Resolver {
	cfg := d.effectiveConfig()
	return aiprovider.NewResolver(d.Pool, d.openRouterClient(), aiprovider.ResolverConfig{
		AbstractionEnabled: cfg.AiProviderAbstractionEnabled,
		PlatformAPIKey:     cfg.OpenRouterAPIKey,
		SecretsKey:         cfg.PlatformSecretsKey,
	})
}

// orgIDPtrForUser resolves the caller's org for provider BYOK scope (nil if unknown).
func (d Deps) orgIDPtrForUser(ctx context.Context, userID uuid.UUID) *uuid.UUID {
	if d.Pool == nil || userID == uuid.Nil {
		return nil
	}
	oid, err := organization.OrgIDForUser(ctx, d.Pool, userID)
	if err != nil {
		return nil
	}
	id := oid
	return &id
}

// aiConfigured reports whether any AI backend can serve requests for the given org scope.
// When abstraction is off, this is equivalent to a platform OpenRouter key/client.
func (d Deps) aiConfigured(ctx context.Context, orgID *uuid.UUID) bool {
	providers := d.aiProvidersConfigured(ctx, orgID)
	return len(providers) > 0
}

// aiProvidersConfigured lists provider names that have usable credentials for the scope.
func (d Deps) aiProvidersConfigured(ctx context.Context, orgID *uuid.UUID) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}

	cfg := d.effectiveConfig()
	if d.openRouterClient() != nil || strings.TrimSpace(cfg.OpenRouterAPIKey) != "" {
		add(string(aiprovider.ProviderOpenRouter))
	}

	if d.Pool == nil {
		return out
	}

	collect := func(scope string, scopeOrg *uuid.UUID) {
		creds, err := aiprovidercreds.ListByScope(ctx, d.Pool, scope, scopeOrg)
		if err != nil {
			return
		}
		for _, c := range creds {
			if !c.Enabled || !c.SecretConfigured {
				continue
			}
			add(c.Provider)
		}
	}

	collect(aiprovidercreds.ScopePlatform, nil)
	if orgID != nil {
		collect(aiprovidercreds.ScopeOrg, orgID)
		if p := legacyTenantBYOKProvider(ctx, d.Pool, *orgID); p != "" {
			add(p)
		}
	}
	return out
}

func legacyTenantBYOKProvider(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) string {
	row, err := tenantaisettings.GetByOrgID(ctx, pool, orgID)
	if err != nil || row == nil || strings.TrimSpace(row.BYOKSecretRef) == "" {
		return ""
	}
	return strings.TrimSpace(row.Provider)
}

// completeStreamOrBuffered streams when the provider supports it; otherwise completes
// and flushes the full text as a single chunk (AP.4 FR-7 buffered fallback).
func (d Deps) completeStreamOrBuffered(
	ctx context.Context,
	orgID *uuid.UUID,
	modelOverride string,
	messages []aiprovider.Message,
	onChunk aiprovider.ChunkHandler,
	opts ...aiprovider.ChatOptions,
) (aiprovider.ChatResult, aiprovider.CallMeta, error) {
	resolver := d.aiProviderResolver()
	result, meta, err := resolver.CompleteStream(ctx, orgID, modelOverride, messages, onChunk, opts...)
	if err == nil || !errors.Is(err, aiprovider.ErrNotSupported) {
		return result, meta, err
	}
	result, meta, err = resolver.Complete(ctx, orgID, modelOverride, messages, opts...)
	if err != nil {
		return result, meta, err
	}
	if onChunk != nil && result.Text != "" {
		if cerr := onChunk(result.Text); cerr != nil {
			return result, meta, cerr
		}
	}
	return result, meta, nil
}
