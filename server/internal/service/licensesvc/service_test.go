package licensesvc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	licenserepo "github.com/lextures/lextures/server/internal/repos/license"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestUtilizationPercent(t *testing.T) {
	if got := UtilizationPercent(8, 10); got != 80 {
		t.Fatalf("got %v want 80", got)
	}
	if got := UtilizationPercent(5, -1); got != 0 {
		t.Fatalf("unlimited got %v", got)
	}
}

func TestCheckCanActivate_Unlimited_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	orgID := organization.SeedDefaultOrgID
	_, _ = pool.Exec(ctx, `DELETE FROM tenant.licenses WHERE org_id = $1`, orgID)
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	em := "seat-unlim-" + time.Now().Format("150405") + "@e.com"
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	uid := uuid.MustParse(row.ID)

	svc := New(pool, config.Config{SeatManagementEnabled: true})
	if err := svc.CheckCanActivate(ctx, uid, orgID); err != nil {
		t.Fatalf("unlimited should pass: %v", err)
	}
}

func TestCheckCanActivate_AtLimit_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	orgID := organization.SeedDefaultOrgID
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	em1 := "seat-a-" + time.Now().Format("150405") + "@e.com"
	u1, err := user.InsertUser(ctx, pool, em1, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := licenserepo.RefreshUsedSeats(ctx, pool, orgID); err != nil {
		t.Fatal(err)
	}
	used, err := licenserepo.CountLearnerSeats(ctx, pool, orgID)
	if err != nil {
		t.Fatal(err)
	}
	max := used
	_, err = licenserepo.Upsert(ctx, pool, orgID, licenserepo.Patch{MaxSeats: &max, Tier: strPtr("starter")})
	if err != nil {
		t.Fatal(err)
	}

	em2 := "seat-b-" + time.Now().Format("150405") + "@e.com"
	u2, err := user.InsertUser(ctx, pool, em2, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	uid2 := uuid.MustParse(u2.ID)
	_ = u1

	svc := New(pool, config.Config{SeatManagementEnabled: true})
	if err := svc.CheckCanActivate(ctx, uid2, orgID); err != ErrSeatLimitReached {
		t.Fatalf("want seat limit, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
