package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestImpersonation_RoundTrip_Pg(t *testing.T) {
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

	emAdmin := "imp-admin-" + time.Now().Format("20060102150405") + "@e.com"
	adminRow, err := user.InsertUser(ctx, pool, emAdmin, ph, nil)
	if err != nil {
		t.Fatalf("admin user: %v", err)
	}
	adminID := uuid.MustParse(adminRow.ID)
	if _, err := orgroles.Create(ctx, pool, defOrg, adminID, nil, orgroles.RoleOrgAdmin, nil, nil); err != nil {
		t.Fatalf("grant org_admin: %v", err)
	}
	slugAdmin, err := organization.OrgSlugForUser(ctx, pool, adminID)
	if err != nil {
		t.Fatal(err)
	}

	emStudent := "imp-student-" + time.Now().Format("20060102150405") + "@e.com"
	studentRow, err := user.InsertUser(ctx, pool, emStudent, ph, nil)
	if err != nil {
		t.Fatalf("student user: %v", err)
	}
	studentID := uuid.MustParse(studentRow.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, studentID, "Student"); err != nil {
		t.Fatalf("student role: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	adminTok, err := signer.Sign(ctx, adminRow.ID, emAdmin, defOrg.String(), slugAdmin, nil)
	if err != nil {
		t.Fatalf("sign admin: %v", err)
	}

	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			AdminConsoleEnabled:  true,
			ImpersonationEnabled: true,
			AdminAuditLogEnabled: true,
		},
	})

	// Start impersonation
	startBody, _ := json.Marshal(map[string]string{"target_user_id": studentRow.ID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin-console/impersonate", bytes.NewReader(startBody))
	req.Header.Set("Authorization", "Bearer "+adminTok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("start status %d: %s", rec.Code, rec.Body.String())
	}
	var startResp struct {
		Token string `json:"impersonation_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &startResp); err != nil || startResp.Token == "" {
		t.Fatalf("start body: %s err=%v", rec.Body.String(), err)
	}

	// GET /me as target
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+startResp.Token)
	meRec := httptest.NewRecorder()
	h.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status %d: %s", meRec.Code, meRec.Body.String())
	}
	var me struct {
		ID            string `json:"id"`
		Impersonating *struct {
			AdminID string `json:"adminId"`
		} `json:"impersonating"`
	}
	if err := json.Unmarshal(meRec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.ID != studentRow.ID || me.Impersonating == nil || me.Impersonating.AdminID != adminRow.ID {
		t.Fatalf("me: %#v", me)
	}

	// POST blocked during impersonation
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/me/push-subscriptions", bytes.NewReader([]byte(`{}`)))
	postReq.Header.Set("Authorization", "Bearer "+startResp.Token)
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	h.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusForbidden {
		t.Fatalf("write block status %d: %s", postRec.Code, postRec.Body.String())
	}

	// Exit impersonation
	endReq := httptest.NewRequest(http.MethodDelete, "/api/v1/admin-console/impersonate/session", nil)
	endReq.Header.Set("Authorization", "Bearer "+startResp.Token)
	endRec := httptest.NewRecorder()
	h.ServeHTTP(endRec, endReq)
	if endRec.Code != http.StatusNoContent {
		t.Fatalf("end status %d: %s", endRec.Code, endRec.Body.String())
	}

	// Impersonation token no longer valid
	meReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq2.Header.Set("Authorization", "Bearer "+startResp.Token)
	meRec2 := httptest.NewRecorder()
	h.ServeHTTP(meRec2, meReq2)
	if meRec2.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token status %d", meRec2.Code)
	}

	// Admin session unchanged
	adminMe := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	adminMe.Header.Set("Authorization", "Bearer "+adminTok)
	adminRec := httptest.NewRecorder()
	h.ServeHTTP(adminRec, adminMe)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin me status %d", adminRec.Code)
	}
	var adminProfile struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(adminRec.Body.Bytes(), &adminProfile)
	if adminProfile.ID != adminRow.ID {
		t.Fatalf("admin profile: %#v", adminProfile)
	}
}
