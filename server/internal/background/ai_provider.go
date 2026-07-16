package background

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/openrouter"
)

// platformScopedCompleter builds a platform-scoped (no org) AI completer for background
// sweeps that run across all tenants (parity with the legacy platform OpenRouter client).
func platformScopedCompleter(pool *pgxpool.Pool, cfg config.Config) aiprovider.ScopedCompleter {
	var orClient *openrouter.Client
	if cfg.OpenRouterAPIKey != "" {
		orClient = openrouter.NewClient(cfg.OpenRouterAPIKey)
	}
	resolver := aiprovider.NewResolver(pool, orClient, aiprovider.ResolverConfig{
		AbstractionEnabled: cfg.AiProviderAbstractionEnabled,
		PlatformAPIKey:     cfg.OpenRouterAPIKey,
		SecretsKey:         cfg.PlatformSecretsKey,
	})
	return aiprovider.BoundCompleter{Resolver: resolver, OrgID: nil}
}
