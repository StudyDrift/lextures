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
	licenserepo "github.com/lextures/lextures/server/internal/repos/license"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestAdminLicense_SeatLimitOnReactivate_Pg(t *testing.T) {
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

	uniq := uuid.NewString()[:8]
	ts := time.Now().Format("20060102150405")
	orgSlug := "sl-" + uniq
	testOrgRow, err := organization.Create(ctx, pool, "Seat Limit Test "+ts, orgSlug, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	testOrg := testOrgRow.ID
	defer func() { _, _ = pool.Exec(ctx, `DELETE FROM tenant.licenses WHERE org_id = $1`, testOrg) }()

	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	emGA := "lic-ga-" + uniq + "@e.com"
	gaRow, err := user.InsertUser(ctx, pool, emGA, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	gaID := uuid.MustParse(gaRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, gaID, "Global Admin"); err != nil {
		t.Fatalf("ga: %v", err)
	}

	emAdmin := "lic-admin-" + uniq + "@e.com"
	adminRow, err := user.InsertUser(ctx, pool, emAdmin, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	adminID := uuid.MustParse(adminRow.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, testOrg, adminID); err != nil {
		t.Fatalf("move admin: %v", err)
	}
	if _, err := orgroles.Create(ctx, pool, testOrg, adminID, nil, orgroles.RoleOrgAdmin, &gaID, nil); err != nil {
		t.Fatalf("grant: %v", err)
	}
	slugAdmin := testOrgRow.Slug

	emA := "lic-a-" + uniq + "@e.com"
	rowA, err := user.InsertUser(ctx, pool, emA, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	idA := uuid.MustParse(rowA.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, testOrg, idA); err != nil {
		t.Fatalf("move user a: %v", err)
	}

	emB := "lic-b-" + uniq + "@e.com"
	rowB, err := user.InsertUser(ctx, pool, emB, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	idB := uuid.MustParse(rowB.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, testOrg, idB); err != nil {
		t.Fatalf("move user b: %v", err)
	}

	_, err = pool.Exec(ctx, `
UPDATE "user".users SET deactivated_at = NOW(), login_blocked = TRUE WHERE id = $1
`, idB)
	if err != nil {
		t.Fatal(err)
	}
	if err := licenserepo.RefreshUsedSeats(ctx, pool, testOrg); err != nil {
		t.Fatal(err)
	}

	used, err := licenserepo.CountLearnerSeats(ctx, pool, testOrg)
	if err != nil {
		t.Fatal(err)
	}
	if used < 1 {
		t.Fatalf("expected at least one active learner seat, got %d", used)
	}
	max := used
	_, err = licenserepo.Upsert(ctx, pool, testOrg, licenserepo.Patch{MaxSeats: &max, Tier: func() *string { s := "starter"; return &s }()})
	if err != nil {
		t.Fatal(err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	adminTok, err := signer.Sign(ctx, adminRow.ID, emAdmin, testOrg.String(), slugAdmin, nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			AdminConsoleEnabled:   true,
			SeatManagementEnabled: true,
		},
	})

	body := []byte(`{"active":true}`)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/admin-console/users/"+idB.String(), bytes.NewReader(body))
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+adminTok)
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reactivate at limit status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("SEAT_LIMIT_REACHED")) {
		t.Fatalf("expected SEAT_LIMIT_REACHED body=%s", rr.Body.String())
	}

	_, err = pool.Exec(ctx, `
UPDATE "user".users SET deactivated_at = NOW(), login_blocked = TRUE WHERE id = $1
`, idA)
	if err != nil {
		t.Fatal(err)
	}
	if err := licenserepo.RefreshUsedSeats(ctx, pool, testOrg); err != nil {
		t.Fatal(err)
	}

	rr2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodPatch, "/api/v1/admin-console/users/"+idB.String(), bytes.NewReader(body))
	r2 = r2.WithContext(ctx)
	r2.Header.Set("Authorization", "Bearer "+adminTok)
	r2.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr2, r2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("reactivate with free seat status=%d body=%s", rr2.Code, rr2.Body.String())
	}
}