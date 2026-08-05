package coursestructure

import (
	"context"
	"fmt"
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

	em := "cstruct-reorder-" + time.Now().Format("20060102150405") + "@e.com"
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
	cc := fmt.Sprintf("C-R%05d", time.Now().UnixNano()%100000)
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
