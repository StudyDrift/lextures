package platformconfig_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/platformconfig"
)

func TestUpsert_WithCourseMarketplaceColumn(t *testing.T) {
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

	// Default ON when unset (AC-1 is covered by merge tests; here we flip off then on).
	off := false
	_, err = platformconfig.Upsert(ctx, pool, &platformconfig.Write{
		FFCourseMarketplace: &off,
	})
	if err != nil {
		t.Fatalf("Upsert off: %v", err)
	}
	row, err := platformconfig.Get(ctx, pool)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row == nil || row.FFCourseMarketplace == nil || *row.FFCourseMarketplace {
		t.Fatal("expected FFCourseMarketplace false in DB")
	}
	merged := platformconfig.Merge(config.Config{}, row)
	if merged.FFCourseMarketplace {
		t.Fatal("expected merged false")
	}

	on := true
	_, err = platformconfig.Upsert(ctx, pool, &platformconfig.Write{
		FFCourseMarketplace: &on,
	})
	if err != nil {
		t.Fatalf("Upsert on: %v", err)
	}
	row, err = platformconfig.Get(ctx, pool)
	if err != nil {
		t.Fatalf("Get after on: %v", err)
	}
	merged = platformconfig.Merge(config.Config{}, row)
	if !merged.FFCourseMarketplace {
		t.Fatal("expected merged true after enable")
	}
}
