package coursestructure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestPermanentlyDeleteArchivedChild_Pg(t *testing.T) {
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

	em := "trash-" + time.Now().Format("20060102150405.000") + "@e.com"
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
	cc := fmt.Sprintf("C-T%05d", time.Now().UnixNano()%100000)
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Trash', $2) RETURNING id
`, cc, uid).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	var modID, pageID, gradedID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'module', 'M', NULL, true, false) RETURNING id
`, courseID).Scan(&modID); err != nil {
		t.Fatalf("module: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 0, 'content_page', 'Gone', $2, true, true) RETURNING id
`, courseID, modID).Scan(&pageID); err != nil {
		t.Fatalf("page: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 1, 'assignment', 'Kept until confirm', $2, true, true) RETURNING id
`, courseID, modID).Scan(&gradedID); err != nil {
		t.Fatalf("assignment: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_grades (course_id, student_user_id, module_item_id, points_earned)
VALUES ($1, $2, $3, 4)
`, courseID, uid, gradedID); err != nil {
		t.Fatalf("grade: %v", err)
	}

	if err := PermanentlyDeleteArchivedChild(ctx, pool, courseID, pageID, false); err != nil {
		t.Fatalf("delete page: %v", err)
	}
	var pageLeft int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.course_structure_items WHERE id = $1`, pageID).Scan(&pageLeft); err != nil {
		t.Fatalf("count page: %v", err)
	}
	if pageLeft != 0 {
		t.Fatalf("page still present")
	}

	err = PermanentlyDeleteArchivedChild(ctx, pool, courseID, gradedID, false)
	if !errors.Is(err, ErrStudentWork) {
		t.Fatalf("expected ErrStudentWork, got %v", err)
	}
	var gradeLeft int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.course_grades WHERE module_item_id = $1`, gradedID).Scan(&gradeLeft); err != nil {
		t.Fatalf("count grade: %v", err)
	}
	if gradeLeft != 1 {
		t.Fatalf("grade removed without confirmation")
	}

	if err := PermanentlyDeleteArchivedChild(ctx, pool, courseID, gradedID, true); err != nil {
		t.Fatalf("confirmed delete: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.course_grades WHERE module_item_id = $1`, gradedID).Scan(&gradeLeft); err != nil {
		t.Fatalf("count grade after: %v", err)
	}
	if gradeLeft != 0 {
		t.Fatalf("grade still in gradebook")
	}
	var itemLeft int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.course_structure_items WHERE id = $1`, gradedID).Scan(&itemLeft); err != nil {
		t.Fatalf("count item: %v", err)
	}
	if itemLeft != 0 {
		t.Fatalf("assignment still present")
	}

	err = PermanentlyDeleteArchivedChild(ctx, pool, courseID, modID, true)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("module delete: got %v", err)
	}

	var liveID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items
    (course_id, sort_order, kind, title, parent_id, published, archived)
    VALUES ($1, 2, 'content_page', 'Live', $2, true, false) RETURNING id
`, courseID, modID).Scan(&liveID); err != nil {
		t.Fatalf("live page: %v", err)
	}
	err = PermanentlyDeleteArchivedChild(ctx, pool, courseID, liveID, true)
	if !errors.Is(err, ErrNotInTrash) {
		t.Fatalf("live item: got %v", err)
	}
}
