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
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// TestAdminAIEndpoints_GlobalAdminWithoutOrgAdminGrant_Pg covers #790: a Global Admin (who can
// reach the AI governance and provider settings pages) must be able to load and save their
// organization's AI config without an explicit org_admin grant, while a plain member stays forbidden.
func TestAdminAIEndpoints_GlobalAdminWithoutOrgAdminGrant_Pg(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	uniq := uuid.NewString()[:8]
	orgRow, err := organization.Create(ctx, pool, "AI Admin Access "+uniq, "ai-admin-"+uniq, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	email := "ai-admin-" + uniq + "@e.com"
	u, err := user.InsertUser(ctx, pool, email, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	userID := uuid.MustParse(u.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, orgRow.ID, userID); err != nil {
		t.Fatalf("move user: %v", err)
	}

	secret := "ai-admin-access-jwt-secret-min32chars"
	signer := auth.NewJWTSigner(secret)
	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			AiDisclosureEnabled:          true,
			AiProviderAbstractionEnabled: true,
			JWTSecret:                    secret,
		},
	})
	tok, err := signer.Sign(ctx, userID.String(), email, orgRow.ID.String(), orgRow.Slug, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	do := func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var buf bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&buf).Encode(body); err != nil {
				t.Fatalf("encode: %v", err)
			}
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Authorization", "Bearer "+tok)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	paths := []string{"/api/v1/admin/ai-config", "/api/v1/admin/ai-settings"}
	for _, p := range paths {
		if w := do(http.MethodGet, p, nil); w.Code != http.StatusForbidden {
			t.Fatalf("member GET %s: status %d %s", p, w.Code, w.Body.String())
		}
	}

	if err := rbac.AssignUserRoleByName(ctx, pool, userID, "Global Admin"); err != nil {
		t.Fatalf("assign Global Admin: %v", err)
	}
	for _, p := range paths {
		if w := do(http.MethodGet, p, nil); w.Code != http.StatusOK {
			t.Fatalf("global admin GET %s: status %d %s", p, w.Code, w.Body.String())
		}
	}

	if w := do(http.MethodPut, "/api/v1/admin/ai-config", map[string]any{
		"featuresEnabled": map[string]bool{"ai_tutor": false},
	}); w.Code != http.StatusOK {
		t.Fatalf("global admin PUT ai-config: status %d %s", w.Code, w.Body.String())
	}
	w := do(http.MethodGet, "/api/v1/admin/ai-config", nil)
	var got struct {
		OrgID           string          `json:"orgId"`
		FeaturesEnabled map[string]bool `json:"featuresEnabled"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if got.OrgID != orgRow.ID.String() {
		t.Fatalf("orgId: %q want %q", got.OrgID, orgRow.ID)
	}
	if v, ok := got.FeaturesEnabled["ai_tutor"]; !ok || v {
		t.Fatalf("featuresEnabled: %v", got.FeaturesEnabled)
	}
}
