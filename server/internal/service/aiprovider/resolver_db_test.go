package aiprovider

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/aiprovidercreds"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

func resolverTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return ctx, pool
}

// TestResolver_OrgCredentialWinsOverPlatform covers AP.9 FR-2 resolution order:
// org BYOK → platform credential for the same provider.
func TestResolver_OrgCredentialWinsOverPlatform(t *testing.T) {
	ctx, pool := resolverTestPool(t)
	secretsKey := make([]byte, 32)
	if _, err := rand.Read(secretsKey); err != nil {
		t.Fatal(err)
	}
	slug := "ai-resolve-" + uuid.NewString()[:8]
	org, err := organization.Create(ctx, pool, "AI Resolve Org", slug, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	orgID := org.ID
	provider := string(ProviderAnthropic)

	if err := aiprovidercreds.Upsert(ctx, pool, aiprovidercreds.ScopePlatform, nil, provider, aiprovidercreds.UpsertInput{
		SetSettings: true,
		Settings:    map[string]any{},
	}); err != nil {
		t.Fatalf("platform upsert: %v", err)
	}
	if err := aiprovidercreds.StoreSecret(ctx, pool, aiprovidercreds.ScopePlatform, nil, provider, secretsKey, "sk-platform-anthropic"); err != nil {
		t.Fatalf("platform secret: %v", err)
	}
	if err := aiprovidercreds.Upsert(ctx, pool, aiprovidercreds.ScopeOrg, &orgID, provider, aiprovidercreds.UpsertInput{
		SetSettings: true,
		Settings:    map[string]any{},
	}); err != nil {
		t.Fatalf("org upsert: %v", err)
	}
	if err := aiprovidercreds.StoreSecret(ctx, pool, aiprovidercreds.ScopeOrg, &orgID, provider, secretsKey, "sk-org-anthropic"); err != nil {
		t.Fatalf("org secret: %v", err)
	}

	r := NewResolver(pool, nil, ResolverConfig{
		AbstractionEnabled: true,
		SecretsKey:         secretsKey,
	})
	auth, err := r.authMaterialForProvider(ctx, &orgID, ProviderAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if auth.APIKey != "sk-org-anthropic" {
		t.Fatalf("want org key, got %q", auth.APIKey)
	}

	// Platform scope alone still resolves the platform key.
	platformAuth, err := r.authMaterialForProvider(ctx, nil, ProviderAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	if platformAuth.APIKey != "sk-platform-anthropic" {
		t.Fatalf("want platform key, got %q", platformAuth.APIKey)
	}
}
