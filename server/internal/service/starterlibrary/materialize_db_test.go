package starterlibrary

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

func TestCopyForUser_PublishesOutline_Pg(t *testing.T) {
	pool := starterTestPool(t)
	ctx := context.Background()
	userID := uuid.New()
	email := "starter-" + userID.String() + "@example.com"
	if _, err := pool.Exec(ctx, `
INSERT INTO "user".users (id, email, password_hash, display_name)
VALUES ($1, $2, 'x', 'Starter Parent')
`, userID, email); err != nil {
		t.Fatalf("user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM "user".users WHERE id = $1`, userID)
	})

	outline, ok := Get("ela-3-5")
	if !ok {
		t.Fatal("missing outline")
	}
	copied, err := CopyForUser(ctx, pool, userID, outline)
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	t.Cleanup(func() {
		_ = course.DeleteByCode(context.Background(), pool, copied.CourseCode)
	})

	pub, err := course.GetPublicByCourseCode(ctx, pool, copied.CourseCode)
	if err != nil || pub == nil {
		t.Fatalf("load: %v", err)
	}
	if !pub.Published {
		t.Fatal("expected published course")
	}
	if len(pub.GradeLevels) != 1 || pub.GradeLevels[0] != Band35 {
		t.Fatalf("grades: %#v", pub.GradeLevels)
	}
	rows, err := coursestructure.ListForCourse(ctx, pool, copied.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	modules, pages := 0, 0
	for _, row := range rows {
		if !row.Published {
			t.Fatalf("unpublished item %s", row.Title)
		}
		switch row.Kind {
		case "module":
			modules++
		case "content_page":
			pages++
		}
	}
	if modules != len(outline.Modules) || pages < modules {
		t.Fatalf("structure modules=%d pages=%d", modules, pages)
	}
}

func starterTestPool(t *testing.T) *pgxpool.Pool {
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
	t.Cleanup(pool.Close)
	return pool
}
