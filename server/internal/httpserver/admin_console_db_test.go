package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestAdminConsole_Overview_OrgAdmin_Pg(t *testing.T) {
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

	emGA := "ac-ga-" + time.Now().Format("20060102150405") + "@e.com"
	gaRow, err := user.InsertUser(ctx, pool, emGA, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	gaID := uuid.MustParse(gaRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, gaID, "Global Admin"); err != nil {
		t.Fatalf("ga: %v", err)
	}
	slugGA, err := organization.OrgSlugForUser(ctx, pool, gaID)
	if err != nil {
		t.Fatal(err)
	}

	emOrg := "ac-org-" + time.Now().Format("20060102150405") + "@e.com"
	orgRow, err := user.InsertUser(ctx, pool, emOrg, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	orgUID := uuid.MustParse(orgRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, orgUID, "Student"); err != nil {
		t.Fatalf("student: %v", err)
	}
	slugOrg, err := organization.OrgSlugForUser(ctx, pool, orgUID)
	if err != nil {
		t.Fatal(err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	gaTok, err := signer.Sign(ctx, gaRow.ID, emGA, defOrg.String(), slugGA, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AdminConsoleEnabled: true, AdminAuditLogEnabled: true},
	})

	grantBody := []byte(`{"userId":"` + orgRow.ID + `","role":"org_admin"}`)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/"+defOrg.String()+"/role-grants", bytes.NewReader(grantBody))
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+gaTok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusCreated {
		t.Fatalf("grant status=%d body=%s", rr.Code, rr.Body.String())
	}

	orgTok, err := signer.Sign(ctx, orgRow.ID, emOrg, defOrg.String(), slugOrg, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	rr2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/overview", nil)
	r2 = r2.WithContext(ctx)
	r2.Header.Set("Authorization", "Bearer "+orgTok)
	h.ServeHTTP(rr2, r2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("overview status=%d body=%s", rr2.Code, rr2.Body.String())
	}

	studentEm := "ac-stu-" + time.Now().Format("20060102150405") + "@e.com"
	stuRow, err := user.InsertUser(ctx, pool, studentEm, ph, nil)
	if err != nil {
		t.Fatalf("student: %v", err)
	}
	stuID := uuid.MustParse(stuRow.ID)
	stuTok, err := signer.Sign(ctx, stuRow.ID, studentEm, defOrg.String(), slugOrg, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	rr3 := httptest.NewRecorder()
	r3 := httptest.NewRequest(http.MethodGet, "/api/v1/admin-console/overview", nil)
	r3 = r3.WithContext(ctx)
	r3.Header.Set("Authorization", "Bearer "+stuTok)
	h.ServeHTTP(rr3, r3)
	if rr3.Code != http.StatusForbidden {
		t.Fatalf("student overview status=%d want 403", rr3.Code)
	}

	_, err = orgroles.UserHasRole(ctx, pool, orgUID, defOrg, orgroles.RoleOrgAdmin)
	if err != nil {
		t.Fatal(err)
	}
	_ = stuID
}
