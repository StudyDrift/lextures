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

// TestApplyModuleAndChildOrder_TopLevelNonModuleNoCollision ensures reordering modules
// does not collide with top-level non-module rows (e.g. attendance) on the unique
// (course_id, sort_order) index for parent_id IS NULL.
func TestApplyModuleAndChildOrder_TopLevelNonModuleNoCollision(t *testing.T) {
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

	em := "cstruct-reorder-top-" + uuid.New().String() + "@e.com"
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
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Reorder top-level', $2) RETURNING id
`, cc, uid).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.courses WHERE id = $1`, courseID)
	})

	// Top-level attendance at sort_order 0 (same unique index as modules).
	var attID, m1, h1 uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'attendance', 'Week 1', NULL, true, false) RETURNING id
`, courseID).Scan(&attID); err != nil {
		t.Fatalf("attendance: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 1, 'module', 'M1', NULL, true, false) RETURNING id
`, courseID).Scan(&m1); err != nil {
		t.Fatalf("m1: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'heading', 'Reading', $2, true, false) RETURNING id
`, courseID, m1).Scan(&h1); err != nil {
		t.Fatalf("h1: %v", err)
	}

	// Same-module child reorder (the user action: move last item above others).
	err = ApplyModuleAndChildOrder(ctx, pool, courseID, []uuid.UUID{m1}, map[uuid.UUID][]uuid.UUID{
		m1: {h1},
	})
	if err != nil {
		t.Fatalf("reorder with top-level attendance: %v", err)
	}

	var modSort, attSort int
	if err := pool.QueryRow(ctx, `SELECT sort_order FROM course.course_structure_items WHERE id = $1`, m1).Scan(&modSort); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT sort_order FROM course.course_structure_items WHERE id = $1`, attID).Scan(&attSort); err != nil {
		t.Fatal(err)
	}
	if modSort != 0 {
		t.Fatalf("module sort_order: got %d want 0", modSort)
	}
	if attSort == modSort {
		t.Fatalf("attendance and module share sort_order %d", attSort)
	}
	if attSort < 1 {
		t.Fatalf("attendance sort_order should be after modules, got %d", attSort)
	}
}
