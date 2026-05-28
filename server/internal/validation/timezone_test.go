package validation

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestValidIANATimezone_KnownAndUnknown(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	ok, err := ValidIANATimezone(ctx, pool, "Asia/Kolkata")
	if err != nil {
		t.Fatalf("valid tz: %v", err)
	}
	if !ok {
		t.Fatal("expected Asia/Kolkata to be valid")
	}

	ok, err = ValidIANATimezone(ctx, pool, "Not/A_Real_Zone")
	if err != nil {
		t.Fatalf("invalid tz: %v", err)
	}
	if ok {
		t.Fatal("expected unknown timezone to be rejected")
	}
}

func TestNormalizeTimezone(t *testing.T) {
	raw := "  America/New_York  "
	got := NormalizeTimezone(&raw)
	if got == nil || *got != "America/New_York" {
		t.Fatalf("got %v", got)
	}
	empty := "   "
	if NormalizeTimezone(&empty) != nil {
		t.Fatal("expected nil for blank")
	}
}
