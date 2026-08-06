package coursestructure

import (
	"context"
	"fmt"
	"strings"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// TestApplyModuleAndChildOrder_MoveBetweenModules_Pg verifies children can change parent modules.
func TestApplyModuleAndChildOrder_MoveBetweenModules_Pg(t *testing.T) {
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

	em := "cstruct-reorder-" + uuid.New().String() + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	var courseID uuid.UUID
	cc := fmt.Sprintf("C-%s", strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6]))
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Reorder', $2) RETURNING id
`, cc, uid).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}

	var m1, m2, a1, a2, b1 uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'module', 'M1', NULL, true, false) RETURNING id
`, courseID).Scan(&m1); err != nil {
		t.Fatalf("m1: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 1, 'module', 'M2', NULL, true, false) RETURNING id
`, courseID).Scan(&m2); err != nil {
		t.Fatalf("m2: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'assignment', 'A1', $2, true, false) RETURNING id
`, courseID, m1).Scan(&a1); err != nil {
		t.Fatalf("a1: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 1, 'assignment', 'A2', $2, true, false) RETURNING id
`, courseID, m1).Scan(&a2); err != nil {
		t.Fatalf("a2: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'content_page', 'B1', $2, true, false) RETURNING id
`, courseID, m2).Scan(&b1); err != nil {
		t.Fatalf("b1: %v", err)
	}

	// Move A1 from M1 to M2 (before B1).
	err = ApplyModuleAndChildOrder(ctx, pool, courseID, []uuid.UUID{m1, m2}, map[uuid.UUID][]uuid.UUID{
		m1: {a2},
		m2: {a1, b1},
	})
	if err != nil {
		t.Fatalf("reorder move: %v", err)
	}

	var parent uuid.UUID
	var sortOrder int
	if err := pool.QueryRow(ctx, `
SELECT parent_id, sort_order FROM course.course_structure_items WHERE id = $1
`, a1).Scan(&parent, &sortOrder); err != nil {
		t.Fatalf("select a1: %v", err)
	}
	if parent != m2 {
		t.Fatalf("a1 parent: got %s want %s", parent, m2)
	}
	if sortOrder != 0 {
		t.Fatalf("a1 sort_order: got %d want 0", sortOrder)
	}

	if err := pool.QueryRow(ctx, `
SELECT parent_id, sort_order FROM course.course_structure_items WHERE id = $1
`, b1).Scan(&parent, &sortOrder); err != nil {
		t.Fatalf("select b1: %v", err)
	}
	if parent != m2 || sortOrder != 1 {
		t.Fatalf("b1 parent/order: %s/%d want m2/1", parent, sortOrder)
	}

	if err := pool.QueryRow(ctx, `
SELECT parent_id, sort_order FROM course.course_structure_items WHERE id = $1
`, a2).Scan(&parent, &sortOrder); err != nil {
		t.Fatalf("select a2: %v", err)
	}
	if parent != m1 || sortOrder != 0 {
		t.Fatalf("a2 parent/order: %s/%d want m1/0", parent, sortOrder)
	}

	// Same-module reorder still works.
	err = ApplyModuleAndChildOrder(ctx, pool, courseID, []uuid.UUID{m1, m2}, map[uuid.UUID][]uuid.UUID{
		m1: {a2},
		m2: {b1, a1},
	})
	if err != nil {
		t.Fatalf("same-module reorder: %v", err)
	}
	if err := pool.QueryRow(ctx, `
SELECT sort_order FROM course.course_structure_items WHERE id = $1
`, a1).Scan(&sortOrder); err != nil {
		t.Fatalf("select a1 after reorder: %v", err)
	}
	if sortOrder != 1 {
		t.Fatalf("a1 sort after same-module reorder: got %d want 1", sortOrder)
	}
}

// TestApplyModuleAndChildOrder_ArchivedChildrenNoCollision_Pg ensures reordering live
// children does not collide with archived siblings on idx_course_structure_items_child_order.
func TestApplyModuleAndChildOrder_ArchivedChildrenNoCollision_Pg(t *testing.T) {
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

	em := "cstruct-reorder-arch-" + uuid.New().String() + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	var courseID uuid.UUID
	cc := fmt.Sprintf("C-%s", strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6]))
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Reorder archived', $2) RETURNING id
`, cc, uid).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.courses WHERE id = $1`, courseID)
	})

	var m1, m2, live1, live2, live3, arch1, arch2 uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'module', 'M1', NULL, true, false) RETURNING id
`, courseID).Scan(&m1); err != nil {
		t.Fatalf("m1: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 1, 'module', 'M2', NULL, true, false) RETURNING id
`, courseID).Scan(&m2); err != nil {
		t.Fatalf("m2: %v", err)
	}
	// Live + archived interleaved under M1 (archived at 0 and 2 historically).
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'content_page', 'Arch1', $2, true, true) RETURNING id
`, courseID, m1).Scan(&arch1); err != nil {
		t.Fatalf("arch1: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 1, 'content_page', 'Live1', $2, true, false) RETURNING id
`, courseID, m1).Scan(&live1); err != nil {
		t.Fatalf("live1: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 2, 'content_page', 'Arch2', $2, true, true) RETURNING id
`, courseID, m1).Scan(&arch2); err != nil {
		t.Fatalf("arch2: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 3, 'content_page', 'Live2', $2, true, false) RETURNING id
`, courseID, m1).Scan(&live2); err != nil {
		t.Fatalf("live2: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'content_page', 'Live3', $2, true, false) RETURNING id
`, courseID, m2).Scan(&live3); err != nil {
		t.Fatalf("live3: %v", err)
	}

	// Same-module reorder of live children (swap) with archived present.
	err = ApplyModuleAndChildOrder(ctx, pool, courseID, []uuid.UUID{m1, m2}, map[uuid.UUID][]uuid.UUID{
		m1: {live2, live1},
		m2: {live3},
	})
	if err != nil {
		t.Fatalf("same-module reorder with archived: %v", err)
	}

	var sortOrder int
	var parent uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT parent_id, sort_order FROM course.course_structure_items WHERE id = $1
`, live2).Scan(&parent, &sortOrder); err != nil {
		t.Fatal(err)
	}
	if parent != m1 || sortOrder != 0 {
		t.Fatalf("live2: parent/order %s/%d want m1/0", parent, sortOrder)
	}
	if err := pool.QueryRow(ctx, `
SELECT parent_id, sort_order FROM course.course_structure_items WHERE id = $1
`, live1).Scan(&parent, &sortOrder); err != nil {
		t.Fatal(err)
	}
	if parent != m1 || sortOrder != 1 {
		t.Fatalf("live1: parent/order %s/%d want m1/1", parent, sortOrder)
	}
	// Archived compacted after live sequence (sort_order >= 2).
	for _, id := range []uuid.UUID{arch1, arch2} {
		if err := pool.QueryRow(ctx, `
SELECT sort_order FROM course.course_structure_items WHERE id = $1
`, id).Scan(&sortOrder); err != nil {
			t.Fatal(err)
		}
		if sortOrder < 2 {
			t.Fatalf("archived %s sort_order %d should be >= 2", id, sortOrder)
		}
	}

	// Cross-module move with archived still under source.
	err = ApplyModuleAndChildOrder(ctx, pool, courseID, []uuid.UUID{m1, m2}, map[uuid.UUID][]uuid.UUID{
		m1: {live1},
		m2: {live2, live3},
	})
	if err != nil {
		t.Fatalf("cross-module move with archived: %v", err)
	}
	if err := pool.QueryRow(ctx, `
SELECT parent_id, sort_order FROM course.course_structure_items WHERE id = $1
`, live2).Scan(&parent, &sortOrder); err != nil {
		t.Fatal(err)
	}
	if parent != m2 || sortOrder != 0 {
		t.Fatalf("live2 after move: parent/order %s/%d want m2/0", parent, sortOrder)
	}

	// Repeat several times to catch temp-value accumulation / collision bugs.
	for i := 0; i < 5; i++ {
		err = ApplyModuleAndChildOrder(ctx, pool, courseID, []uuid.UUID{m1, m2}, map[uuid.UUID][]uuid.UUID{
			m1: {live1},
			m2: {live3, live2},
		})
		if err != nil {
			t.Fatalf("repeat reorder %d: %v", i, err)
		}
		err = ApplyModuleAndChildOrder(ctx, pool, courseID, []uuid.UUID{m2, m1}, map[uuid.UUID][]uuid.UUID{
			m1: {live1},
			m2: {live2, live3},
		})
		if err != nil {
			t.Fatalf("repeat reorder modules %d: %v", i, err)
		}
	}
}
