package migrate

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
)

// TestRun_FullMigrations_Integration runs the SQL files when DATABASE_URL is set (CI, local).
func TestRun_FullMigrations_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("use full go test to exercise migrations with Postgres")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run integration test")
	}
	if err := RunWithFS(context.Background(), serverdata.Migrations, dsn); err != nil {
		t.Fatal(err)
	}
	if err := RunWithFS(context.Background(), serverdata.Migrations, dsn); err != nil {
		t.Fatalf("second run: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer p.Close()
	if err := FromPool(ctx, serverdata.Migrations, p); err != nil {
		t.Fatalf("from pool: %v", err)
	}
}

// TestRollbackLatest_Integration rolls back the latest migration with a real down.sql and re-applies.
func TestRollbackLatest_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("use full go test to exercise migrations with Postgres")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("set DATABASE_URL to run integration test")
	}
	ctx := context.Background()
	if err := RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatal(err)
	}

	err := RollbackLatest(ctx, serverdata.Migrations, dsn)
	if err != nil {
		if errors.Is(err, ErrRollbackNotSupported) {
			t.Skip("latest migration has no executable down.sql")
		}
		t.Fatal(err)
	}

	if err := RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("re-apply after rollback: %v", err)
	}
}

// TestLint_EmbeddedMigrations ensures every up migration has a companion down.sql.
func TestLint_EmbeddedMigrations(t *testing.T) {
	res, err := LintFS(serverdata.Migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("lint errors:\n%s", FormatLintReport(res))
	}
}
