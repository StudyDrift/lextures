package platformconfig_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/platformconfig"
)

func TestUpsert_WithEnrollmentStateMachineColumn(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	v := true
	_, err = platformconfig.Upsert(ctx, pool, &platformconfig.Write{
		H5PEnabled: &v,
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	merged := platformconfig.Merge(config.Load(), nil)
	row, err := platformconfig.Get(ctx, pool)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil {
		t.Fatal("expected row")
	}
	merged = platformconfig.Merge(merged, row)
	if !merged.FFEnrollmentStateMachine {
		t.Fatalf("expected FFEnrollmentStateMachine true, got false")
	}
}
