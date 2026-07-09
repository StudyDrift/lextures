package marketplacecourses

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
)

func TestEnsureProvisioned_CreateAndIdempotent_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	resetHarnessCourse(t, pool)

	first, err := svc.EnsureProvisioned(ctx, "harness-smoke")
	if err != nil {
		t.Fatalf("first provision: %v", err)
	}
	if !first.Created {
		t.Fatal("expected created on fresh database")
	}
	if first.Report.Skipped {
		t.Fatal("expected content sync on first provision")
	}

	var published, listed, official bool
	var price int
	var mode string
	var openEnroll bool
	err = pool.QueryRow(ctx, `
SELECT published, marketplace_listed, is_official, price_cents, course_mode::text, open_enrollment
FROM course.courses WHERE id = $1
`, first.ID).Scan(&published, &listed, &official, &price, &mode, &openEnroll)
	if err != nil {
		t.Fatal(err)
	}
	if !published || !listed || !official || price != 0 || mode != "self_paced" || !openEnroll {
		t.Fatalf("course flags: published=%v listed=%v official=%v price=%d mode=%s open=%v",
			published, listed, official, price, mode, openEnroll)
	}

	var moduleCount, pageCount, quizCount, assignCount int
	if err := pool.QueryRow(ctx, `
SELECT
  COUNT(*) FILTER (WHERE kind = 'module' AND parent_id IS NULL AND NOT archived),
  COUNT(*) FILTER (WHERE kind = 'content_page' AND NOT archived),
  COUNT(*) FILTER (WHERE kind = 'quiz' AND NOT archived),
  COUNT(*) FILTER (WHERE kind = 'assignment' AND NOT archived)
FROM course.course_structure_items WHERE course_id = $1
`, first.ID).Scan(&moduleCount, &pageCount, &quizCount, &assignCount); err != nil {
		t.Fatal(err)
	}
	if moduleCount != 1 || pageCount < 1 || quizCount != 1 || assignCount < 1 {
		t.Fatalf("structure modules=%d pages=%d quizzes=%d assigns=%d", moduleCount, pageCount, quizCount, assignCount)
	}

	var sectionCount int
	if err := pool.QueryRow(ctx, `
SELECT jsonb_array_length(COALESCE(cs.sections, '[]'::jsonb))
FROM course.course_syllabus cs WHERE cs.course_id = $1
`, first.ID).Scan(&sectionCount); err != nil {
		t.Fatal(err)
	}
	if sectionCount < 1 {
		t.Fatalf("syllabus sections: %d", sectionCount)
	}

	// Capture structure item id for a page before re-provision.
	var pageItemID uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT structure_item_id FROM settings.marketplace_course_items
WHERE course_slug = 'harness-smoke' AND slug = 'm1.welcome.what-is-this'
`).Scan(&pageItemID); err != nil {
		t.Fatal(err)
	}

	second, err := svc.EnsureProvisioned(ctx, "harness-smoke")
	if err != nil {
		t.Fatalf("second provision: %v", err)
	}
	if second.Created {
		t.Fatal("expected reconcile, not create")
	}
	if first.ID != second.ID {
		t.Fatalf("course id changed: %s vs %s", first.ID, second.ID)
	}
	if !second.Report.Skipped {
		t.Fatalf("expected second sync noop, got %+v", second.Report)
	}

	var pageItemID2 uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT structure_item_id FROM settings.marketplace_course_items
WHERE course_slug = 'harness-smoke' AND slug = 'm1.welcome.what-is-this'
`).Scan(&pageItemID2); err != nil {
		t.Fatal(err)
	}
	if pageItemID != pageItemID2 {
		t.Fatalf("structure_item_id changed on noop: %s vs %s", pageItemID, pageItemID2)
	}

	var count int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.courses WHERE short_code = 'LEX-MC-SMOKE'
`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one smoke course, got %d", count)
	}
}

func TestEnsureProvisioned_ContentVersionResync_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	resetHarnessCourse(t, pool)

	first, err := svc.EnsureProvisioned(ctx, "harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	var pageItemID uuid.UUID
	var oldMarkdown string
	if err := pool.QueryRow(ctx, `
SELECT mci.structure_item_id, mcp.markdown
FROM settings.marketplace_course_items mci
INNER JOIN course.module_content_pages mcp ON mcp.structure_item_id = mci.structure_item_id
WHERE mci.course_slug = 'harness-smoke' AND mci.slug = 'm1.welcome.what-is-this'
`).Scan(&pageItemID, &oldMarkdown); err != nil {
		t.Fatal(err)
	}

	// Simulate a content_version bump + body edit without changing the structure item id.
	if _, err := pool.Exec(ctx, `
UPDATE settings.marketplace_course_items
SET content_version = 0, updated_at = NOW()
WHERE course_slug = 'harness-smoke' AND slug = 'm1.welcome.what-is-this'
`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE course.module_content_pages
SET markdown = 'stale body', updated_at = NOW()
WHERE structure_item_id = $1
`, pageItemID); err != nil {
		t.Fatal(err)
	}

	second, err := svc.EnsureProvisioned(ctx, "harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	if second.Report.Skipped {
		t.Fatal("expected resync after content_version mismatch")
	}

	var pageItemID2 uuid.UUID
	var newMarkdown string
	var version int
	if err := pool.QueryRow(ctx, `
SELECT mci.structure_item_id, mci.content_version, mcp.markdown
FROM settings.marketplace_course_items mci
INNER JOIN course.module_content_pages mcp ON mcp.structure_item_id = mci.structure_item_id
WHERE mci.course_slug = 'harness-smoke' AND mci.slug = 'm1.welcome.what-is-this'
`).Scan(&pageItemID2, &version, &newMarkdown); err != nil {
		t.Fatal(err)
	}
	if pageItemID != pageItemID2 {
		t.Fatalf("structure_item_id must be stable: %s vs %s", pageItemID, pageItemID2)
	}
	if version != 1 {
		t.Fatalf("expected content_version restored to 1, got %d", version)
	}
	if newMarkdown == "stale body" || newMarkdown == "" {
		t.Fatalf("expected body restored from fixtures, got %q", newMarkdown)
	}
	if newMarkdown != oldMarkdown {
		// Body should match original fixture content again.
		t.Fatalf("markdown mismatch after resync")
	}
	_ = first
}

func TestEnsureProvisioned_Concurrent_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	resetHarnessCourse(t, pool)

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := New(pool)
			if _, err := svc.EnsureProvisioned(ctx, "harness-smoke"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent provision: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.courses WHERE short_code = 'LEX-MC-SMOKE'
`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one course after concurrent provision, got %d", count)
	}
}

func TestEnsureProvisioned_DoesNotResetEnrollmentCount_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	resetHarnessCourse(t, pool)

	c, err := svc.EnsureProvisioned(ctx, "harness-smoke")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE course.courses SET enrollment_count = 42, average_rating = 4.50 WHERE id = $1
`, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EnsureProvisioned(ctx, "harness-smoke"); err != nil {
		t.Fatal(err)
	}
	var enroll int
	var rating *float64
	if err := pool.QueryRow(ctx, `
SELECT enrollment_count, average_rating FROM course.courses WHERE id = $1
`, c.ID).Scan(&enroll, &rating); err != nil {
		t.Fatal(err)
	}
	if enroll != 42 {
		t.Fatalf("enrollment_count reset: %d", enroll)
	}
	if rating == nil || *rating != 4.5 {
		t.Fatalf("average_rating reset: %v", rating)
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
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
	t.Cleanup(func() { pool.Close() })
	return pool
}

func resetHarnessCourse(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `DELETE FROM course.courses WHERE short_code = 'LEX-MC-SMOKE'`); err != nil {
		t.Fatalf("reset harness course: %v", err)
	}
	// Ledger rows cascade from course delete; also clear orphaned ledger if any.
	if _, err := pool.Exec(ctx, `DELETE FROM settings.marketplace_courses WHERE slug = 'harness-smoke'`); err != nil {
		t.Fatalf("reset ledger: %v", err)
	}
}
