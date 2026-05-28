package test

import (
	"bytes"
	"context"
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
	repo "github.com/lextures/lextures/server/internal/repos/aidisclosure"
	"github.com/lextures/lextures/server/internal/repos/user"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
)

type aiDisclosureEnv struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
}

func setupAIDisclosure(t *testing.T) *aiDisclosureEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping AI disclosure e2e tests")
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

	secret := "ai-disclosure-e2e-jwt-secret-min32chars"
	signer := auth.NewJWTSigner(secret)
	email := fmt.Sprintf("ai-disclosure-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!ai-disclosure")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "AI Disclosure Test"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			AiDisclosureEnabled: true,
			JWTSecret:           secret,
			OpenRouterAPIKey:    "test-key-not-used",
		},
	}))
	t.Cleanup(srv.Close)

	return &aiDisclosureEnv{srv: srv, pool: pool, signer: signer, userID: userID}
}

func (e *aiDisclosureEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), "ai-e2e@test.example", "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func aiDo(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
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
		t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return resp
}

func TestAIDisclosure_OptOutBlocksNotebookQuery(t *testing.T) {
	env := setupAIDisclosure(t)
	tok := env.token(t)
	ctx := context.Background()

	if err := repo.SetOptOut(ctx, env.pool, env.userID, true); err != nil {
		t.Fatalf("set opt out: %v", err)
	}
	aigateway.InvalidateOptOutCache(env.userID)

	resp := aiDo(t, env.srv, http.MethodPost, "/api/v1/me/notebooks/query", map[string]any{
		"question":  "test",
		"notebooks": []any{},
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("notebook query: status=%d want 403", resp.StatusCode)
	}
	var errBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody.Error.Code != "AI_PROCESSING_DISABLED" {
		t.Fatalf("code=%q want AI_PROCESSING_DISABLED", errBody.Error.Code)
	}

	h := aigateway.UserIDHash("ai-disclosure-e2e-jwt-secret-min32chars", env.userID)
	rows, err := repo.ListLogsByUserHash(ctx, env.pool, h, 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(rows) == 0 || !rows[0].Blocked {
		t.Fatal("expected blocked inference log row")
	}
}

func TestAIDisclosure_PublicDisclosureJSON(t *testing.T) {
	env := setupAIDisclosure(t)
	resp := aiDo(t, env.srv, http.MethodGet, "/api/v1/public/ai-disclosure", nil, "")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var doc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc["models"] == nil {
		t.Fatal("expected models in disclosure doc")
	}
}

func TestAIDisclosure_OptOutToggle(t *testing.T) {
	env := setupAIDisclosure(t)
	tok := env.token(t)

	resp := aiDo(t, env.srv, http.MethodPut, "/api/v1/settings/ai-opt-out", map[string]bool{
		"aiProcessingOptOut": true,
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put opt-out: status=%d", resp.StatusCode)
	}

	resp2 := aiDo(t, env.srv, http.MethodGet, "/api/v1/settings/ai-opt-out", nil, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("get opt-out: status=%d", resp2.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["aiProcessingOptOut"] != true {
		t.Fatalf("opt out=%v want true", got["aiProcessingOptOut"])
	}
}
