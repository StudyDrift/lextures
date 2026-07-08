package introcourse

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
	"github.com/lextures/lextures/server/internal/repos/license"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

func TestEnsureProvisioned_Idempotent_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true}
	resetIntroCourse(t, pool, svc)

	first, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatalf("first provision: %v", err)
	}
	if !first.Created {
		t.Fatal("expected created on fresh database")
	}

	second, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatalf("second provision: %v", err)
	}
	if second.Created {
		t.Fatal("expected reconcile, not create")
	}
	if first.ID != second.ID {
		t.Fatalf("course id changed: %s vs %s", first.ID, second.ID)
	}

	var count int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.courses WHERE short_code = $1
`, ShortCode).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one intro course, got %d", count)
	}

	var published bool
	var teacherID uuid.UUID
	err = pool.QueryRow(ctx, `
SELECT c.published, ce.user_id
FROM course.courses c
INNER JOIN course.course_enrollments ce ON ce.course_id = c.id AND ce.role = 'teacher'
WHERE c.short_code = $1
`, ShortCode).Scan(&published, &teacherID)
	if err != nil {
		t.Fatal(err)
	}
	if !published {
		t.Fatal("expected published=true")
	}
	if teacherID != SystemUserID {
		t.Fatalf("expected system instructor %s, got %s", SystemUserID, teacherID)
	}

	var heroURL *string
	var heroPosition *string
	err = pool.QueryRow(ctx, `
SELECT hero_image_url, hero_image_object_position
FROM course.courses
WHERE short_code = $1
`, ShortCode).Scan(&heroURL, &heroPosition)
	if err != nil {
		t.Fatal(err)
	}
	if heroURL == nil || !strings.Contains(*heroURL, "/course-files/") {
		t.Fatalf("expected intro course hero banner URL, got %#v", heroURL)
	}
	if heroPosition == nil || strings.TrimSpace(*heroPosition) != introHeroObjectPosition {
		t.Fatalf("expected hero object position %q, got %#v", introHeroObjectPosition, heroPosition)
	}
}

func TestEnsureProvisioned_DisabledSkipsCreate_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)

	resetIntroCourse(t, pool, svc)

	_, err := svc.EnsureProvisioned(ctx, config.Config{IntroCourseEnabled: false})
	if err != nil {
		t.Fatalf("provision disabled: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.courses WHERE short_code = $1`, ShortCode).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected no course when disabled on empty DB, got %d", count)
	}
}

func TestEnsureProvisioned_ConcurrentSingleCourse_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	cfg := config.Config{IntroCourseEnabled: true}

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc := New(pool)
			svc.InvalidateCache()
			if _, err := svc.EnsureProvisioned(ctx, cfg); err != nil {
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
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.courses WHERE short_code = $1`, ShortCode).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one course after concurrent provision, got %d", count)
	}
}

func TestSystemUser_ExcludedFromSeatsAndPeople_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	if _, err := svc.EnsureProvisioned(ctx, config.Config{IntroCourseEnabled: true}); err != nil {
		t.Fatal(err)
	}

	used, err := license.CountLearnerSeats(ctx, pool, organization.SeedDefaultOrgID)
	if err != nil {
		t.Fatal(err)
	}
	var includesGuide int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM "user".users
WHERE id = $1 AND account_type = 'system'
`, icrepo.SystemUserID).Scan(&includesGuide); err != nil {
		t.Fatal(err)
	}
	if includesGuide != 1 {
		t.Fatal("expected guide system user present")
	}
	// Seat count must not include any system accounts.
	var systemInSeats int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM "user".users u
WHERE u.org_id = $1
  AND u.account_type = 'system'
  AND u.deactivated_at IS NULL
  AND NOT u.login_blocked
`, organization.SeedDefaultOrgID).Scan(&systemInSeats); err != nil {
		t.Fatal(err)
	}
	if systemInSeats == 0 {
		t.Fatal("expected at least the guide system user in org")
	}
	_ = used // used reflects human learners only; system users are excluded by migration 359.
}

func TestCourseID_CacheInvalidation_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	resetIntroCourse(t, pool, svc)

	if _, ok, err := svc.CourseID(ctx); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("expected no course before provision")
	}

	if _, err := svc.EnsureProvisioned(ctx, config.Config{IntroCourseEnabled: true}); err != nil {
		t.Fatal(err)
	}
	id1, ok, err := svc.CourseID(ctx)
	if err != nil || !ok || id1 == uuid.Nil {
		t.Fatalf("CourseID after provision: ok=%v err=%v id=%s", ok, err, id1)
	}
	svc.InvalidateCache()
	id2, ok, err := svc.CourseID(ctx)
	if err != nil || !ok || id2 != id1 {
		t.Fatalf("CourseID after cache invalidation: %v %v %s", ok, err, id2)
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

// resetIntroCourse removes the canonical intro course so tests can assert create vs reconcile
// on a shared CI database where other packages may have provisioned it first.
func resetIntroCourse(t *testing.T, pool *pgxpool.Pool, svc *Service) {
	t.Helper()
	ctx := context.Background()
	svc.InvalidateCache()
	if _, err := pool.Exec(ctx, `DELETE FROM course.courses WHERE short_code = $1`, ShortCode); err != nil {
		t.Fatalf("reset intro course: %v", err)
	}
}