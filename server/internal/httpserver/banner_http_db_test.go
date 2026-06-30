package httpserver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func bannerTestSetup(t *testing.T) (*pgxpool.Pool, *auth.JWTSigner, string, uuid.UUID, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	em := "banner-ga-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	gaID, _ := uuid.Parse(row.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, gaID, "Global Admin"); err != nil {
		t.Fatalf("role: %v", err)
	}
	orgID, err := organization.OrgIDForUser(ctx, pool, gaID)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return pool, signer, tok, gaID, orgID
}

func TestMaintenanceBanner_CreateListPublicDelete_Pg(t *testing.T) {
	pool, signer, tok, _, orgID := bannerTestSetup(t)
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			MaintenanceBannerEnabled: true,
			AdminConsoleEnabled:      true,
		},
	})

	createBody := map[string]any{
		"scope":    "org",
		"message":  "Maintenance at midnight",
		"severity": "warning",
	}
	bodyBytes, _ := json.Marshal(createBody)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/banners", bytes.NewReader(bodyBytes))
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}

	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/status/banner", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("public status=%d", w2.Code)
	}
	var pub map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &pub); err != nil {
		t.Fatalf("public json: %v", err)
	}
	if pub["message"] != "Maintenance at midnight" {
		t.Fatalf("public message=%v", pub["message"])
	}

	otherEmail := "banner-other-" + time.Now().Format("20060102150405.000") + "@e.com"
	other, err := user.InsertUser(context.Background(), pool, otherEmail, mustHash(t), nil)
	if err != nil {
		t.Fatalf("other user: %v", err)
	}
	otherID, _ := uuid.Parse(other.ID)
	otherOrgRow, err := organization.Create(context.Background(), pool, "Other Org", "other-"+uuid.NewString()[:8], nil, nil, "", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE "user".users SET org_id = $1 WHERE id = $2`, otherOrgRow.ID, otherID); err != nil {
		t.Fatalf("assign org: %v", err)
	}

	otherTok, err := signer.Sign(context.Background(), other.ID, otherEmail, "", "", nil)
	if err != nil {
		t.Fatalf("other sign: %v", err)
	}
	r3 := httptest.NewRequest(http.MethodGet, "/api/v1/status/banner", nil)
	r3.Header.Set("Authorization", "Bearer "+otherTok)
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, r3)
	var otherPub any
	_ = json.Unmarshal(w3.Body.Bytes(), &otherPub)
	if otherPub != nil {
		t.Fatalf("other org should not see banner, got %v", otherPub)
	}

	var created map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id, _ := created["id"].(string)
	del := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/banners/"+id, nil)
	del.Header.Set("Authorization", "Bearer "+tok)
	wDel := httptest.NewRecorder()
	h.ServeHTTP(wDel, del)
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d", wDel.Code)
	}

	r4 := httptest.NewRequest(http.MethodGet, "/api/v1/status/banner", nil)
	r4.Header.Set("Authorization", "Bearer "+tok)
	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, r4)
	var after any
	_ = json.Unmarshal(w4.Body.Bytes(), &after)
	if after != nil {
		t.Fatalf("expected null after delete, got %v", after)
	}
	_ = orgID
}

func TestMaintenanceBanner_Expiry_Pg(t *testing.T) {
	pool, signer, tok, _, _ := bannerTestSetup(t)
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			MaintenanceBannerEnabled: true,
			AdminConsoleEnabled:      true,
		},
	})
	exp := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	createBody := map[string]any{
		"scope":     "global",
		"message":   "Expired notice",
		"severity":  "info",
		"expiresAt": exp,
	}
	bodyBytes, _ := json.Marshal(createBody)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/banners", bytes.NewReader(bodyBytes))
	r.Header.Set("Authorization", "Bearer "+tok)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/status/banner", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	var pub any
	_ = json.Unmarshal(w2.Body.Bytes(), &pub)
	if pub != nil {
		t.Fatalf("expired banner should not be active, got %v", pub)
	}
}

func mustHash(t *testing.T) string {
	t.Helper()
	h, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestMaintenanceBanner_OrgAdminScope_Pg(t *testing.T) {
	pool, signer, gaTok, _, defOrg := bannerTestSetup(t)
	ctx := context.Background()
	orgAdminEmail := "banner-oa-" + time.Now().Format("20060102150405.000") + "@e.com"
	oa, err := user.InsertUser(ctx, pool, orgAdminEmail, mustHash(t), nil)
	if err != nil {
		t.Fatalf("oa user: %v", err)
	}
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			MaintenanceBannerEnabled: true,
			AdminConsoleEnabled:      true,
		},
	})
	grantBody := []byte(`{"userId":"` + oa.ID + `","role":"org_admin"}`)
	rr := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/"+defOrg.String()+"/role-grants", bytes.NewReader(grantBody))
	rr.Header.Set("Authorization", "Bearer "+gaTok)
	rr.Header.Set("Content-Type", "application/json")
	wGrant := httptest.NewRecorder()
	h.ServeHTTP(wGrant, rr)
	if wGrant.Code != http.StatusCreated {
		t.Fatalf("grant status=%d body=%s", wGrant.Code, wGrant.Body.String())
	}
	oaTok, err := signer.Sign(ctx, oa.ID, orgAdminEmail, "", "", nil)
	if err != nil {
		t.Fatalf("sign oa: %v", err)
	}
	bodyBytes, _ := json.Marshal(map[string]any{
		"scope":    "org",
		"message":  "District SSO migration Monday",
		"severity": "info",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/banners", bytes.NewReader(bodyBytes))
	r.Header.Set("Authorization", "Bearer "+oaTok)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("org admin create status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestMaintenanceBanner_StatuspageWebhook_Pg(t *testing.T) {
	pool, signer, _, _, _ := bannerTestSetup(t)
	secret := "statuspage-test-secret"
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			MaintenanceBannerEnabled: true,
			StatuspageWebhookSecret:  secret,
		},
	})
	payload := []byte(`{"incident":{"id":"inc-1","name":"API degraded","status":"investigating","impact":"major"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/banners/statuspage-webhook", bytes.NewReader(payload))
	r.Header.Set("X-Statuspage-Signature", sig)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("webhook status=%d body=%s", w.Code, w.Body.String())
	}
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/status/banner", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	var pub map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &pub); err != nil {
		t.Fatalf("json: %v", err)
	}
	if pub["severity"] != "error" {
		t.Fatalf("severity=%v want error", pub["severity"])
	}
}
