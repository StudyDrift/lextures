package platformpeople

import (
	"context"
	"os"
	"testing"
	"time"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
)

const humanUserFilter = `
  u.account_type <> 'system'
  AND u.email NOT ILIKE '%@erased.invalid'
`

func TestFetchDashboardStats_MatchesDirectCounts_Pg(t *testing.T) {
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

	stats, err := FetchDashboardStats(ctx, pool)
	if err != nil {
		t.Fatalf("FetchDashboardStats: %v", err)
	}

	var direct struct {
		total, active, suspended, signups7d, recent30d int64
	}
	err = pool.QueryRow(ctx, `
SELECT
    COUNT(*)::bigint,
    COUNT(*) FILTER (WHERE u.deactivated_at IS NULL AND NOT u.login_blocked)::bigint,
    COUNT(*) FILTER (WHERE u.deactivated_at IS NOT NULL OR u.login_blocked)::bigint,
    COUNT(*) FILTER (WHERE u.created_at >= NOW() - INTERVAL '7 days')::bigint
FROM "user".users u
WHERE `+humanUserFilter).Scan(
		&direct.total, &direct.active, &direct.suspended, &direct.signups7d,
	)
	if err != nil {
		t.Fatalf("direct user counts: %v", err)
	}
	err = pool.QueryRow(ctx, `
SELECT COUNT(DISTINCT ua.user_id)::bigint
FROM "user".user_audit ua
INNER JOIN "user".users u ON u.id = ua.user_id
WHERE ua.occurred_at >= NOW() - INTERVAL '30 days'
  AND `+humanUserFilter).Scan(&direct.recent30d)
	if err != nil {
		t.Fatalf("direct recent activity: %v", err)
	}

	if stats.TotalAccounts != direct.total {
		t.Fatalf("totalAccounts = %d, want %d", stats.TotalAccounts, direct.total)
	}
	if stats.ActiveAccounts != direct.active {
		t.Fatalf("activeAccounts = %d, want %d", stats.ActiveAccounts, direct.active)
	}
	if stats.SuspendedAccounts != direct.suspended {
		t.Fatalf("suspendedAccounts = %d, want %d", stats.SuspendedAccounts, direct.suspended)
	}
	if stats.SignupsLast7Days != direct.signups7d {
		t.Fatalf("signupsLast7Days = %d, want %d", stats.SignupsLast7Days, direct.signups7d)
	}
	if stats.RecentlyActive30Days != direct.recent30d {
		t.Fatalf("recentlyActive30Days = %d, want %d", stats.RecentlyActive30Days, direct.recent30d)
	}
}