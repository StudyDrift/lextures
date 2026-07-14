package platformcourses

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
)

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

	// REPEATABLE READ snapshot so stats + verification counts stay consistent
	// while parallel package tests insert/delete courses on the shared CI database.
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	stats, err := fetchDashboardStats(ctx, tx)
	if err != nil {
		t.Fatalf("FetchDashboardStats: %v", err)
	}

	var direct struct {
		total, active, draft, archived, created7d int64
	}
	err = tx.QueryRow(ctx, `
SELECT
    COUNT(*)::bigint,
    COUNT(*) FILTER (WHERE c.archived = false AND c.published = true)::bigint,
    COUNT(*) FILTER (WHERE c.archived = false AND c.published = false)::bigint,
    COUNT(*) FILTER (WHERE c.archived = true)::bigint,
    COUNT(*) FILTER (WHERE c.created_at >= NOW() - INTERVAL '7 days')::bigint
FROM course.courses c
`).Scan(
		&direct.total, &direct.active, &direct.draft, &direct.archived, &direct.created7d,
	)
	if err != nil {
		t.Fatalf("direct course counts: %v", err)
	}

	if stats.TotalCourses != direct.total {
		t.Fatalf("totalCourses = %d, want %d", stats.TotalCourses, direct.total)
	}
	if stats.ActiveCourses != direct.active {
		t.Fatalf("activeCourses = %d, want %d", stats.ActiveCourses, direct.active)
	}
	if stats.DraftCourses != direct.draft {
		t.Fatalf("draftCourses = %d, want %d", stats.DraftCourses, direct.draft)
	}
	if stats.ArchivedCourses != direct.archived {
		t.Fatalf("archivedCourses = %d, want %d", stats.ArchivedCourses, direct.archived)
	}
	if stats.CreatedLast7Days != direct.created7d {
		t.Fatalf("createdLast7Days = %d, want %d", stats.CreatedLast7Days, direct.created7d)
	}

	// Smoke the public pool entrypoint (non-snapshot); must not error.
	if _, err := FetchDashboardStats(ctx, pool); err != nil {
		t.Fatalf("FetchDashboardStats(pool): %v", err)
	}
}
