package platformcourses

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
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestEnsureAdminAccess_GlobalAdminCrossOrg_Pg(t *testing.T) {
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

	var otherOrg uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO tenant.organizations (slug, name, status)
VALUES ($1, $2, 'active') RETURNING id
`, "ga-access-"+time.Now().Format("150405.000"), "Global Admin Access Org").Scan(&otherOrg); err != nil {
		t.Fatalf("org: %v", err)
	}

	em := "ga-access-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid := uuid.MustParse(row.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, uid, "Global Admin"); err != nil {
		t.Fatalf("ga: %v", err)
	}

	courseCode, err := course.RandomCourseCode()
	if err != nil {
		t.Fatal(err)
	}
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id, org_id, published)
VALUES ($1, 'Other org course', $2, $3, true) RETURNING id
`, courseCode, uid, otherOrg).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}

	ok, err := enrollment.UserHasAccess(ctx, pool, courseCode, uid)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no access before admin enrollment")
	}

	if err := EnsureAdminAccess(ctx, pool, courseID, uid); err != nil {
		t.Fatalf("EnsureAdminAccess: %v", err)
	}

	ok, err = enrollment.UserHasAccess(ctx, pool, courseCode, uid)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected Global Admin to access a course in another org after access grant")
	}

	roles, err := enrollment.UserRolesInCourse(ctx, pool, courseCode, uid)
	if err != nil {
		t.Fatal(err)
	}
	foundTeacher := false
	for _, r := range roles {
		if r == "teacher" {
			foundTeacher = true
			break
		}
	}
	if !foundTeacher {
		t.Fatalf("expected teacher role after access grant, got %v", roles)
	}
}
