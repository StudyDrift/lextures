package aiprovidercreds

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
	"github.com/lextures/lextures/server/internal/repos/organization"
)

func testPool(t *testing.T) (context.Context, *pgxpool.Pool) {
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

func TestStoreDecryptClear_Platform(t *testing.T) {
	ctx, pool := testPool(t)
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	provider := "anthropic"

	if err := Upsert(ctx, pool, ScopePlatform, nil, provider, UpsertInput{
		Settings:    map[string]any{"note": "platform"},
		SetSettings: true,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	secret := "sk-ant-test-platform-key"
	if err := StoreSecret(ctx, pool, ScopePlatform, nil, provider, key, secret); err != nil {
		t.Fatalf("store: %v", err)
	}
	got, err := DecryptSecret(ctx, pool, ScopePlatform, nil, provider, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != secret {
		t.Fatalf("got %q", got)
	}
	resolved, settings, enabled, err := ResolveAPIKey(ctx, pool, ScopePlatform, nil, provider, key, "")
	if err != nil || !enabled || resolved != secret {
		t.Fatalf("resolve: key=%q enabled=%v err=%v", resolved, enabled, err)
	}
	if settings["note"] != "platform" {
		t.Fatalf("settings: %v", settings)
	}
	if err := ClearSecret(ctx, pool, ScopePlatform, nil, provider); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, err = DecryptSecret(ctx, pool, ScopePlatform, nil, provider, key)
	if err != nil || got != "" {
		t.Fatalf("after clear: %q err=%v", got, err)
	}
	_ = Delete(ctx, pool, ScopePlatform, nil, provider)
}

func TestResolveAPIKey_LegacyOpenRouter(t *testing.T) {
	ctx, pool := testPool(t)
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	legacy := "legacy-openrouter-key"
	resolved, _, enabled, err := ResolveAPIKey(ctx, pool, ScopePlatform, nil, "openrouter", key, legacy)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || resolved != legacy {
		t.Fatalf("want legacy dual-read, got key=%q enabled=%v", resolved, enabled)
	}
}

func TestOrgCredential_UniquePerProvider(t *testing.T) {
	ctx, pool := testPool(t)
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	slug := "ai-creds-" + uuid.NewString()[:8]
	org, err := organization.Create(ctx, pool, "AI Creds Org", slug, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	orgID := org.ID
	for _, provider := range []string{"openai", "anthropic"} {
		if err := Upsert(ctx, pool, ScopeOrg, &orgID, provider, UpsertInput{SetSettings: true, Settings: map[string]any{}}); err != nil {
			t.Fatalf("upsert %s: %v", provider, err)
		}
		if err := StoreSecret(ctx, pool, ScopeOrg, &orgID, provider, key, "key-"+provider); err != nil {
			t.Fatalf("store %s: %v", provider, err)
		}
	}
	list, err := ListByScope(ctx, pool, ScopeOrg, &orgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("want >=2 credentials, got %d", len(list))
	}
	openai, err := DecryptSecret(ctx, pool, ScopeOrg, &orgID, "openai", key)
	if err != nil || openai != "key-openai" {
		t.Fatalf("openai: %q err=%v", openai, err)
	}
	anthropic, err := DecryptSecret(ctx, pool, ScopeOrg, &orgID, "anthropic", key)
	if err != nil || anthropic != "key-anthropic" {
		t.Fatalf("anthropic: %q err=%v", anthropic, err)
	}
}
