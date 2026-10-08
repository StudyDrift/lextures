package enrollment

import (
	"context"
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

func TestUserAccessibleCourseIDs_EmptyInput(t *testing.T) {
	got, err := UserAccessibleCourseIDs(context.Background(), nil, uuid.New(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %d", len(got))
	}
}

func TestUserAccessibleCourseIDs_NotEnrolled_Pg(t *testing.T) {
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
	em := "acc-" + time.Now().Format("20060102150405.000000") + "@e.com"
	ph, err := auth.HashPassword("password1230")
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(u.ID)
	other := uuid.New()
	got, err := UserAccessibleCourseIDs(ctx, pool, uid, []uuid.UUID{other})
	if err != nil {
		t.Fatalf("accessible: %v", err)
	}
	if got[other] {
		t.Fatalf("expected course %s to be inaccessible", other)
	}
}
