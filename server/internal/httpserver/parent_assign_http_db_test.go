package httpserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/auth/hibp"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/parentlinks"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestParentAssign_LinkAndInvite_Pg(t *testing.T) {
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

	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ts := time.Now().Format("20060102150405")

	emGA := "pa-ga-" + ts + "@e.com"
	gaRow, err := user.InsertUser(ctx, pool, emGA, ph, nil)
	if err != nil {
		t.Fatalf("ga: %v", err)
	}
	gaID := uuid.MustParse(gaRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, gaID, "Global Admin"); err != nil {
		t.Fatalf("ga role: %v", err)
	}

	emStaff := "pa-staff-" + ts + "@e.com"
	staffRow, err := user.InsertUser(ctx, pool, emStaff, ph, nil)
	if err != nil {
		t.Fatalf("staff: %v", err)
	}
	staffID := uuid.MustParse(staffRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, staffID, "Teacher"); err != nil {
		t.Fatalf("teacher role: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
CROSS JOIN "user".permissions p
WHERE r.name = 'Teacher' AND p.permission_string = 'org:parent-links:assign:manage'
ON CONFLICT DO NOTHING
`); err != nil {
		t.Fatalf("grant perm: %v", err)
	}

	emStu := "pa-stu-" + ts + "@e.com"
	dn := "Sam Student"
	stuRow, err := user.InsertUser(ctx, pool, emStu, ph, &dn)
	if err != nil {
		t.Fatalf("student: %v", err)
	}
	stuID := uuid.MustParse(stuRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, stuID, "Student"); err != nil {
		t.Fatalf("stu role: %v", err)
	}

	emParent := "pa-par-" + ts + "@e.com"
	pdn := "Pat Parent"
	parRow, err := user.InsertUser(ctx, pool, emParent, ph, &pdn)
	if err != nil {
		t.Fatalf("parent: %v", err)
	}
	parID := uuid.MustParse(parRow.ID)

	orgID, err := organization.OrgIDForUser(ctx, pool, gaID)
	if err != nil {
		t.Fatal(err)
	}
	orgSlug, err := organization.OrgSlugForUser(ctx, pool, gaID)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		FFParentPortal:  true,
		PublicWebOrigin: "http://localhost:5173",
		JWTSecret:       "01234567890123456789012345678901",
	}
	signer := auth.NewJWTSignerWithPool(cfg.JWTSecret, pool)
	stub := hibp.StubChecker{Result: hibp.Result{BreachFound: false, HIBPAvailable: true}}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: cfg, PasswordChecker: stub})

	noPermTok, err := signer.Sign(ctx, stuRow.ID, emStu, orgID.String(), orgSlug, nil)
	if err != nil {
		t.Fatal(err)
	}
	r0 := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+orgID.String()+"/parent-assign/students?q=Sam", nil)
	r0 = r0.WithContext(ctx)
	r0.Header.Set("Authorization", "Bearer "+noPermTok)
	w0 := httptest.NewRecorder()
	h.ServeHTTP(w0, r0)
	if w0.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without perm, got %d %s", w0.Code, w0.Body.String())
	}

	staffTok, err := signer.Sign(ctx, staffRow.ID, emStaff, orgID.String(), orgSlug, nil)
	if err != nil {
		t.Fatal(err)
	}

	r1 := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+orgID.String()+"/parent-assign/students?q="+emStu, nil)
	r1.Header.Set("Authorization", "Bearer "+staffTok)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("search: %d %s", w1.Code, w1.Body.String())
	}

	inviteEmail := "pa-invite-" + ts + "@e.com"
	body, _ := json.Marshal(map[string]any{
		"guardians": []map[string]string{
			{"name": "Pat Parent", "email": emParent, "relationship": "parent"},
			{"name": "New Guardian", "email": inviteEmail, "relationship": "guardian"},
		},
	})
	r2 := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/"+orgID.String()+"/parent-assign/students/"+stuID.String()+"/guardians", bytes.NewReader(body))
	r2.Header.Set("Authorization", "Bearer "+staffTok)
	r2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("assign: %d %s", w2.Code, w2.Body.String())
	}
	var assignOut struct {
		Results []struct {
			Email  string `json:"email"`
			Status string `json:"status"`
			LinkID string `json:"linkId"`
		} `json:"results"`
	}
	if err := json.NewDecoder(w2.Body).Decode(&assignOut); err != nil {
		t.Fatal(err)
	}
	if len(assignOut.Results) != 2 {
		t.Fatalf("want 2 results, got %+v", assignOut.Results)
	}
	var linked, invited int
	var inviteLinkID string
	for _, res := range assignOut.Results {
		switch res.Status {
		case "linked":
			linked++
		case "invited":
			invited++
			inviteLinkID = res.LinkID
		default:
			t.Fatalf("unexpected status %+v", res)
		}
	}
	if linked != 1 || invited != 1 {
		t.Fatalf("want 1 linked + 1 invited, got linked=%d invited=%d", linked, invited)
	}

	ln, err := parentlinks.ActiveLinkBetween(ctx, pool, orgID, parID, stuID)
	if err != nil || ln == nil || ln.Status != "active" {
		t.Fatalf("expected active link for existing parent: %+v %v", ln, err)
	}

	r3 := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/"+orgID.String()+"/parent-assign/students/"+stuID.String()+"/links", nil)
	r3.Header.Set("Authorization", "Bearer "+staffTok)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, r3)
	if w3.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w3.Code, w3.Body.String())
	}

	if inviteLinkID == "" {
		t.Fatal("missing invite link id")
	}
	r4 := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/"+orgID.String()+"/parent-assign/links/"+inviteLinkID+"/resend", nil)
	r4.Header.Set("Authorization", "Bearer "+staffTok)
	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, r4)
	if w4.Code != http.StatusOK {
		t.Fatalf("resend: %d %s", w4.Code, w4.Body.String())
	}

	raw := "test-activate-token-" + ts
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])
	linkUUID := uuid.MustParse(inviteLinkID)
	var parentUID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT parent_user_id FROM "user".parent_student_links WHERE id = $1`, linkUUID).Scan(&parentUID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE "user".parent_link_invites
SET token_hash = $2, expires_at = now() + interval '1 hour', consumed_at = NULL
WHERE link_id = $1
`, linkUUID, hash); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	consumeBody, _ := json.Marshal(map[string]string{
		"token":    raw,
		"password": "BrandNewPassword99!",
	})
	r5 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/parent-invite/consume", bytes.NewReader(consumeBody))
	r5.Header.Set("Content-Type", "application/json")
	w5 := httptest.NewRecorder()
	h.ServeHTTP(w5, r5)
	if w5.Code != http.StatusOK {
		t.Fatalf("consume: %d %s", w5.Code, w5.Body.String())
	}
	ln2, err := parentlinks.ActiveLinkBetween(ctx, pool, orgID, parentUID, stuID)
	if err != nil || ln2 == nil || ln2.Status != "active" {
		t.Fatalf("expected active after consume: %+v %v", ln2, err)
	}

	_ = gaID
}
