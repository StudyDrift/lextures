package organization

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
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestCreateWithCreator_AssignsUser_Pg(t *testing.T) {
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

	em := "org-creator-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	urow, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(urow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, uid, "Global Admin"); err != nil {
		t.Fatalf("ga: %v", err)
	}

	slug := "creator-test-" + time.Now().Format("150405")
	row, err := CreateWithCreator(ctx, pool, uid, "Creator Org", slug, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("CreateWithCreator: %v", err)
	}
	got, err := OrgIDForUser(ctx, pool, uid)
	if err != nil {
		t.Fatalf("OrgIDForUser: %v", err)
	}
	if got != row.ID {
		t.Fatalf("org_id = %v, want %v", got, row.ID)
	}
	ok, err := rbac.UserHasPermission(ctx, pool, uid, "global:app:rbac:manage")
	if err != nil {
		t.Fatalf("rbac: %v", err)
	}
	if !ok {
		t.Fatal("expected creator to retain Global Admin permission")
	}
}