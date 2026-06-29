package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestAdminSearch_OrgScoped_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

	defOrg := organization.SeedDefaultOrgID
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ts := time.Now().Format("20060102150405")
	emOrg := "as-org-" + ts + "@e.com"
	aliceName := "Alice Johnson"
	orgRow, err := user.InsertUser(ctx, pool, emOrg, ph, &aliceName)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	orgUID := uuid.MustParse(orgRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, orgUID, "Student"); err != nil {
		t.Fatalf("role: %v", err)
	}
	slugOrg, err := organization.OrgSlugForUser(ctx, pool, orgUID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := orgroles.Create(ctx, pool, defOrg, orgUID, nil, orgroles.RoleOrgAdmin, &orgUID, nil); err != nil {
		t.Fatalf("grant org admin: %v", err)
	}
	_, err = pool.Exec(ctx, `
UPDATE "user".users SET first_name = 'Alice', last_name = 'Johnson' WHERE id = $1
`, orgUID)
	if err != nil {
		t.Fatalf("update alice: %v", err)
	}

	otherOrg, err := organization.Create(ctx, pool, "Other Org "+ts, "other-"+ts, nil, nil, "us-east-1", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	emOther := "as-other-" + ts + "@e.com"
	bobName := "Bob Johnson"
	otherRow, err := user.InsertUser(ctx, pool, emOther, ph, &bobName)
	if err != nil {
		t.Fatalf("other user: %v", err)
	}
	otherUID := uuid.MustParse(otherRow.ID)
	_, err = pool.Exec(ctx, `
UPDATE "user".users SET org_id = $1, first_name = 'Bob', last_name = 'Johnson' WHERE id = $2
`, otherOrg.ID, otherUID)
	if err != nil {
		t.Fatalf("move other user: %v", err)
	}

	_, err = course.CreateCourse(ctx, pool, orgUID, "Biology 101", "Intro biology course", "", nil, nil, nil)
	if err != nil {
		t.Fatalf("course: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	orgTok, err := signer.Sign(ctx, orgRow.ID, emOrg, defOrg.String(), slugOrg, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AdminConsoleEnabled: true, AdminSearchEnabled: true},
	})

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/search?q=johnson&types=users", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+orgTok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("search status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Alice Johnson") && !strings.Contains(body, emOrg) {
		t.Fatalf("expected org user in results: %s", body)
	}
	if strings.Contains(body, emOther) || strings.Contains(body, "Bob Johnson") {
		t.Fatalf("cross-org user leaked: %s", body)
	}

	rr2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/search?q=biologe&types=courses", nil)
	r2 = r2.WithContext(ctx)
	r2.Header.Set("Authorization", "Bearer "+orgTok)
	h.ServeHTTP(rr2, r2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("fuzzy search status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	if !strings.Contains(rr2.Body.String(), "Biology") {
		t.Fatalf("expected Biology course via fuzzy match: %s", rr2.Body.String())
	}
}
