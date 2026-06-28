package publicapi_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/publicapi"
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

// AC-6: request-log rows older than the retention window are deleted; recent
// rows are kept.
func TestDeleteRequestLogsOlderThan(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	now := time.Now().UTC()
	marker := "/retention-test/" + now.Format("150405.000000")

	insert := func(age time.Duration) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO api.request_log (method, path, status, latency_ms, created_at)
			 VALUES ('GET', $1, 200, 1, $2)`, marker, now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	insert(100 * 24 * time.Hour) // older than 90d — should be deleted
	insert(10 * 24 * time.Hour)  // recent — should be kept
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM api.request_log WHERE path = $1`, marker) })

	cutoff := now.Add(-publicapi.RequestLogRetention)
	n, err := publicapi.DeleteRequestLogsOlderThan(ctx, pool, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 deletion, got %d", n)
	}

	var remaining int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM api.request_log WHERE path = $1`, marker).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("expected 1 recent row to remain, got %d", remaining)
	}
}
