package test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	serverdata "github.com/lextures/lextures/server"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/httpserver"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	tenantaisettings "github.com/lextures/lextures/server/internal/repos/tenantaisettings"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
)

type aiProviderEnv struct {
	srv        *httptest.Server
	pool       *pgxpool.Pool
	signer     *auth.JWTSigner
	userID     uuid.UUID
	orgID      uuid.UUID
	orgSlug    string
	email      string
	secretsKey []byte
}

func setupAIProvider(t *testing.T) *aiProviderEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping AI provider e2e tests")
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
	t.Cleanup(pool.Close)

	secretsKey := make([]byte, 32)
	if _, err := rand.Read(secretsKey); err != nil {
		t.Fatalf("secrets key: %v", err)
	}

	secret := "ai-provider-e2e-jwt-secret-min32chars"
	signer := auth.NewJWTSigner(secret)
	email := fmt.Sprintf("ai-provider-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!ai-provider")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "AI Provider Test"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	slug := fmt.Sprintf("ai-prov-%d", time.Now().UnixNano())
	orgRow, err := organization.Create(ctx, pool, "AI Provider Org", slug, nil, nil, "", nil)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	orgID := orgRow.ID
	_, err = pool.Exec(ctx, `UPDATE "user".users SET org_id = $1 WHERE id = $2`, orgID, userID)
	if err != nil {
		t.Fatalf("user org: %v", err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO "user".org_role_grants (org_id, user_id, role)
VALUES ($1, $2, 'org_admin')
ON CONFLICT DO NOTHING
`, orgID, userID)
	if err != nil {
		t.Fatalf("org admin grant: %v", err)
	}

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			AiDisclosureEnabled:          true,
			AiProviderAbstractionEnabled: true,
			JWTSecret:                    secret,
			PlatformSecretsKey:           secretsKey,
		},
	}))
	t.Cleanup(srv.Close)

	return &aiProviderEnv{
		srv: srv, pool: pool, signer: signer, userID: userID, orgID: orgID,
		orgSlug: slug, email: email, secretsKey: secretsKey,
	}
}

func (e *aiProviderEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), e.email, e.orgID.String(), e.orgSlug, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestAIProvider_AdminSettings_CRUD(t *testing.T) {
	env := setupAIProvider(t)
	token := env.token(t)

	getRes := aiProviderDo(t, env.srv, http.MethodGet, "/api/v1/admin/ai-settings", nil, token)
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get status: %d %s", getRes.StatusCode, readBody(t, getRes))
	}
	var got map[string]any
	decodeJSON(t, getRes, &got)
	if got["provider"] != string(aiprovider.ProviderOpenRouter) {
		t.Fatalf("default provider: %v", got["provider"])
	}

	putBody := map[string]any{
		"provider":   "dry_run",
		"modelAlias": "claude-3-5-sonnet",
		"byokApiKey": "tenant-test-key-should-not-appear-in-logs",
	}
	putRes := aiProviderDo(t, env.srv, http.MethodPut, "/api/v1/admin/ai-settings", putBody, token)
	if putRes.StatusCode != http.StatusOK {
		t.Fatalf("put status: %d %s", putRes.StatusCode, readBody(t, putRes))
	}

	row, err := tenantaisettings.GetByOrgID(context.Background(), env.pool, env.orgID)
	if err != nil || row == nil {
		t.Fatalf("row: %v", err)
	}
	if row.Provider != "dry_run" {
		t.Fatalf("provider: %s", row.Provider)
	}

	key, err := tenantaisettings.DecryptBYOK(context.Background(), env.pool, env.orgID, env.secretsKey)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if key != "tenant-test-key-should-not-appear-in-logs" {
		t.Fatalf("unexpected key")
	}

	testRes := aiProviderDo(t, env.srv, http.MethodPost, "/api/v1/admin/ai-settings/test", nil, token)
	if testRes.StatusCode != http.StatusOK {
		t.Fatalf("test status: %d %s", testRes.StatusCode, readBody(t, testRes))
	}
}

func TestAIProvider_FeatureFlagDisabled(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	signer := auth.NewJWTSigner("ai-provider-flag-jwt-secret-min32c")
	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AiProviderAbstractionEnabled: false},
	}))
	t.Cleanup(srv.Close)
	res := aiProviderDo(t, srv, http.MethodGet, "/api/v1/admin/ai-settings", nil, "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status: %d", res.StatusCode)
	}
}

func aiProviderDo(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}
	req, err := http.NewRequest(method, srv.URL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("req: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return res
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer func() { _ = res.Body.Close() }()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(res.Body)
	return buf.String()
}

func decodeJSON(t *testing.T, res *http.Response, out any) {
	t.Helper()
	defer func() { _ = res.Body.Close() }()
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("decode: %v", err)
	}
}