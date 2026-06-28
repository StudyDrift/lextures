package apitokens_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/apitokens"
)

func testPool(t *testing.T) *pgxpool.Pool {
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
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// AC-3: expired and revoked API tokens are deleted; live tokens are kept.
func TestDeleteExpiredAndRevoked(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	now := time.Now().UTC()
	suffix := uuid.NewString()[:8]

	var userID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO "user".users (email, password_hash) VALUES ($1, 'x') RETURNING id`,
		"tok-"+suffix+"@example.test").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, userID) })

	mkToken := func(label string, expiresAt, revokedAt *time.Time) uuid.UUID {
		var id uuid.UUID
		if err := pool.QueryRow(ctx,
			`INSERT INTO auth.api_tokens (owner_user_id, label, token_hash, token_prefix, expires_at, revoked_at)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			userID, label, "hash-"+label+"-"+suffix, "pfx"+suffix[:5], expiresAt, revokedAt).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}

	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	expiredID := mkToken("expired", &past, nil)
	revokedID := mkToken("revoked", nil, &past)
	liveID := mkToken("live", &future, nil)

	exists := func(id uuid.UUID) bool {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM auth.api_tokens WHERE id = $1`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n == 1
	}

	n, err := apitokens.DeleteExpiredAndRevoked(ctx, pool, now)
	if err != nil {
		t.Fatal(err)
	}
	if n < 2 {
		t.Fatalf("expected at least 2 deletions, got %d", n)
	}
	if exists(expiredID) {
		t.Error("expired token should be deleted")
	}
	if exists(revokedID) {
		t.Error("revoked token should be deleted")
	}
	if !exists(liveID) {
		t.Error("live token should be kept")
	}
}
