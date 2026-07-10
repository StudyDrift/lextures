package licensesvc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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

func testPool(t *testing.T) (*pgxpool.Pool, context.Context, context.CancelFunc) {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		cancel()
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		cancel()
		t.Fatalf("pool: %v", err)
	}
	return pool, ctx, cancel
}

// isolatedOrg creates a dedicated org so parallel package tests cannot race on
// SeedDefaultOrgID seat counts / license rows.
func isolatedOrg(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	uniq := uuid.NewString()[:8]
	slug := "licsvc-" + uniq
	row, err := organization.Create(ctx, pool, "LicenseSvc "+uniq, slug, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenant.licenses WHERE org_id = $1`, row.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM "user".users WHERE org_id = $1`, row.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenant.organizations WHERE id = $1`, row.ID)
	})
	return row.ID
}

func insertLearnerInOrg(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, email string) uuid.UUID {
	t.Helper()
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	row, err := user.InsertUser(ctx, pool, email, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	uid := uuid.MustParse(row.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, orgID, uid); err != nil {
		t.Fatalf("move user to org: %v", err)
	}
	return uid
}

func TestCheckCanActivate_Unlimited_Pg(t *testing.T) {
	pool, ctx, cancel := testPool(t)
	defer cancel()
	defer pool.Close()

	orgID := isolatedOrg(t, ctx, pool)
	// No license row → Effective defaults to unlimited.
	uid := insertLearnerInOrg(t, ctx, pool, orgID, "seat-unlim-"+uuid.NewString()[:8]+"@e.com")

	svc := New(pool, config.Config{SeatManagementEnabled: true})
	if err := svc.CheckCanActivate(ctx, uid, orgID); err != nil {
		t.Fatalf("unlimited should pass: %v", err)
	}
}

func TestCheckCanActivate_AtLimit_Pg(t *testing.T) {
	pool, ctx, cancel := testPool(t)
	defer cancel()
	defer pool.Close()

	orgID := isolatedOrg(t, ctx, pool)
	_ = insertLearnerInOrg(t, ctx, pool, orgID, "seat-a-"+uuid.NewString()[:8]+"@e.com")

	if err := licenserepo.RefreshUsedSeats(ctx, pool, orgID); err != nil {
		t.Fatal(err)
	}
	used, err := licenserepo.CountLearnerSeats(ctx, pool, orgID)
	if err != nil {
		t.Fatal(err)
	}
	if used < 1 {
		t.Fatalf("expected at least one learner seat, got %d", used)
	}
	max := used
	if _, err := licenserepo.Upsert(ctx, pool, orgID, licenserepo.Patch{MaxSeats: &max, Tier: strPtr("starter")}); err != nil {
		t.Fatal(err)
	}

	uid2 := insertLearnerInOrg(t, ctx, pool, orgID, "seat-b-"+uuid.NewString()[:8]+"@e.com")

	svc := New(pool, config.Config{SeatManagementEnabled: true})
	if err := svc.CheckCanActivate(ctx, uid2, orgID); err != ErrSeatLimitReached {
		t.Fatalf("want seat limit, got %v", err)
	}
}

func strPtr(s string) *string { return &s }
