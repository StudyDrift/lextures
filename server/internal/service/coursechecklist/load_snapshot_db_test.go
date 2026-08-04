package coursechecklist

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("short")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedChecklistCourse(t *testing.T, pool *pgxpool.Pool, sectionsEnabled bool, withDates bool, structureN int) (courseID uuid.UUID, courseCode string) {
	t.Helper()
	ctx := context.Background()
	courseID = uuid.New()
	courseCode = "C-" + strings.ToUpper(strings.ReplaceAll(courseID.String(), "-", "")[:6])
	var starts, ends any
	if withDates {
		starts = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		ends = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.courses (id, course_code, title, sections_enabled, starts_at, ends_at)
VALUES ($1, $2, 'CC.1 test', $3, $4, $5)
`, courseID, courseCode, sectionsEnabled, starts, ends)
	if err != nil {
		t.Fatalf("seed course: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.courses WHERE id = $1`, courseID)
	})

	if sectionsEnabled {
		_, err = pool.Exec(ctx, `
INSERT INTO course.course_sections (id, course_id, section_code, name, status)
VALUES ($1, $2, 'A', 'Section A', 'active')
`, uuid.New(), courseID)
		if err != nil {
			t.Fatalf("seed section: %v", err)
		}
	}

	if structureN > 0 {
		modID := uuid.New()
		_, err = pool.Exec(ctx, `
INSERT INTO course.course_structure_items (id, course_id, sort_order, kind, title, published, archived)
VALUES ($1, $2, 0, 'module', 'Module 1', true, false)
`, modID, courseID)
		if err != nil {
			t.Fatalf("seed module: %v", err)
		}
		for i := 0; i < structureN; i++ {
			_, err = pool.Exec(ctx, `
INSERT INTO course.course_structure_items (id, course_id, sort_order, kind, title, parent_id, published, archived)
VALUES ($1, $2, $3, 'content_page', $4, $5, true, false)
`, uuid.New(), courseID, i, "Page "+uuid.NewString()[:8], modID)
			if err != nil {
				t.Fatalf("seed item %d: %v", i, err)
			}
		}
	}
	return courseID, courseCode
}

func TestLoadSnapshotQueryBudgetAndMapping(t *testing.T) {
	pool := testPool(t)
	_, code := seedChecklistCourse(t, pool, true, true, 10)

	var n int64
	snap, err := LoadSnapshotCounted(context.Background(), pool, code, AllDataNeeds, &n)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if n > MaxSnapshotQueries || snap.QueryCount > MaxSnapshotQueries {
		t.Fatalf("query count=%d (snap=%d) exceeds budget %d", n, snap.QueryCount, MaxSnapshotQueries)
	}
	if snap.CourseCode != code {
		t.Fatalf("courseCode=%q", snap.CourseCode)
	}
	if snap.StartsAt == nil || snap.EndsAt == nil {
		t.Fatal("expected dates")
	}
	if !snap.SectionsEnabled || len(snap.Sections) != 1 {
		t.Fatalf("sections=%v enabled=%v", snap.Sections, snap.SectionsEnabled)
	}
	if len(snap.StructureItems) < 11 {
		t.Fatalf("structure items=%d", len(snap.StructureItems))
	}
}

func TestLoadSnapshotOnlyModeNeeds(t *testing.T) {
	pool := testPool(t)
	_, code := seedChecklistCourse(t, pool, true, false, 5)
	reg := MustDefault()
	needs := DataNeedsForEvaluate(reg, EvaluateOptions{Only: []ItemID{ItemCourseDates}})
	var n int64
	snap, err := LoadSnapshotCounted(context.Background(), pool, code, needs, &n)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if n != 1 {
		t.Fatalf("Only-mode query count=%d want 1 (course row)", n)
	}
	if len(snap.StructureItems) != 0 || len(snap.Sections) != 0 {
		t.Fatalf("Only-mode loaded unexpected slices: struct=%d sections=%d", len(snap.StructureItems), len(snap.Sections))
	}
	res := Evaluate(context.Background(), snap, EvaluateOptions{Only: []ItemID{ItemCourseDates}})
	if len(res.Findings) != 1 || res.Findings[0].Finding.Status != StatusTodo {
		t.Fatalf("result=%+v", res.Findings)
	}
}

func TestLoadSnapshotEvaluateIntegration(t *testing.T) {
	pool := testPool(t)
	_, code := seedChecklistCourse(t, pool, false, true, 0)
	snap, err := LoadSnapshot(context.Background(), pool, code, DataNeedsForEvaluate(MustDefault(), EvaluateOptions{}))
	if err != nil {
		t.Fatal(err)
	}
	res := Evaluate(context.Background(), snap, EvaluateOptions{})
	if res.Findings[0].Finding.Status != StatusDone {
		t.Fatalf("dates=%s", res.Findings[0].Finding.Status)
	}
	if res.Findings[1].Finding.Status != StatusNotApplicable {
		t.Fatalf("sections=%s", res.Findings[1].Finding.Status)
	}
}

func TestPeopleStubSQLHasNoEmail(t *testing.T) {
	// Guard the enrollment helper SQL against selecting email / birth date columns.
	src, err := os.ReadFile("../../repos/enrollment/role_counts.go")
	if err != nil {
		t.Fatal(err)
	}
	// Extract only the ListPeopleStubsForCourse query body.
	idx := strings.Index(string(src), "func ListPeopleStubsForCourse")
	if idx < 0 {
		t.Fatal("ListPeopleStubsForCourse missing")
	}
	chunk := strings.ToLower(string(src)[idx:])
	if end := strings.Index(chunk, "\nfunc "); end > 0 {
		chunk = chunk[:end]
	}
	for _, banned := range []string{"u.email", "email,", "date_of_birth", ".dob", " birth"} {
		if strings.Contains(chunk, banned) {
			t.Fatalf("ListPeopleStubsForCourse must not select %q", banned)
		}
	}
}

func BenchmarkLoadSnapshotFull(b *testing.B) {
	if testing.Short() {
		b.Skip("short")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		b.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		b.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(pool.Close)

	// Seed a larger fixture once.
	courseID := uuid.New()
	code := "C-" + strings.ToUpper(strings.ReplaceAll(courseID.String(), "-", "")[:6])
	_, err = pool.Exec(ctx, `
INSERT INTO course.courses (id, course_code, title, sections_enabled, starts_at, ends_at)
VALUES ($1, $2, 'CC.1 bench', false, NOW(), NOW() + interval '90 days')
`, courseID, code)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.courses WHERE id = $1`, courseID)
	})
	modID := uuid.New()
	_, err = pool.Exec(ctx, `
INSERT INTO course.course_structure_items (id, course_id, sort_order, kind, title, published, archived)
VALUES ($1, $2, 0, 'module', 'M', true, false)
`, modID, courseID)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 300; i++ {
		_, err = pool.Exec(ctx, `
INSERT INTO course.course_structure_items (id, course_id, sort_order, kind, title, parent_id, published, archived)
VALUES ($1, $2, $3, 'content_page', $4, $5, true, false)
`, uuid.New(), courseID, i, "P", modID)
		if err != nil {
			b.Fatal(err)
		}
	}

	var maxQueries int64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var n int64
		snap, err := LoadSnapshotCounted(ctx, pool, code, AllDataNeeds, &n)
		if err != nil {
			b.Fatal(err)
		}
		_ = Evaluate(ctx, snap, EvaluateOptions{})
		if n > maxQueries {
			maxQueries = n
		}
	}
	b.StopTimer()
	if maxQueries > MaxSnapshotQueries {
		b.Fatalf("query count %d > %d", maxQueries, MaxSnapshotQueries)
	}
	// Blocking regression threshold: p95 < 400ms is environment-dependent;
	// report ns/op via go test -bench and fail only on extreme outliers (>2s avg).
	if b.Elapsed()/time.Duration(b.N) > 2*time.Second {
		b.Fatalf("avg iteration too slow: %v", b.Elapsed()/time.Duration(b.N))
	}
}
