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
	repobackup "github.com/lextures/lextures/server/internal/repos/backup"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

type backupEnv struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
	email  string
}

func setupBackupOps(t *testing.T) *backupEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping backup/restore e2e tests")
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

	signer := auth.NewJWTSigner("backup-e2e-jwt-secret-min32chars-xx")

	email := fmt.Sprintf("backup-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!backup")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "Backup Test User"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	if err := rbac.AssignUserRoleByName(ctx, pool, userID, "Global Admin"); err != nil {
		t.Fatalf("assign Global Admin: %v", err)
	}

	now := time.Now().UTC()
	next := now.Add(24 * time.Hour)
	if err := repobackup.UpsertTierStatus(ctx, pool, repobackup.TierPostgres, &now, intPtr(120), intPtr(30), &next, nil); err != nil {
		t.Fatalf("seed postgres tier: %v", err)
	}
	if err := repobackup.UpsertTierStatus(ctx, pool, repobackup.TierObjectStorage, &now, intPtr(600), nil, &next, nil); err != nil {
		t.Fatalf("seed object_storage tier: %v", err)
	}

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			BackupModuleEnabled: true,
		},
	}))
	t.Cleanup(srv.Close)

	return &backupEnv{srv: srv, pool: pool, signer: signer, userID: userID, email: email}
}

func intPtr(v int) *int { return &v }

func (e *backupEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), e.email, "", "", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func backupDo(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	req, err := http.NewRequest(method, srv.URL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestBackupOps_StatusAndRestoreDrill(t *testing.T) {
	env := setupBackupOps(t)
	tok := env.token(t)

	resp := backupDo(t, env.srv, http.MethodGet, "/api/v1/internal/ops/backup-status", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("backup-status: status=%d want 200", resp.StatusCode)
	}
	var status struct {
		Targets struct {
			PostgresRPOMinutes int `json:"postgresRpoMinutes"`
		} `json:"targets"`
		Tiers []struct {
			Tier          string  `json:"tier"`
			LastSuccessAt *string `json:"lastSuccessAt"`
		} `json:"tiers"`
		RestoreDrills []any `json:"restoreDrills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.Targets.PostgresRPOMinutes != 60 {
		t.Errorf("postgresRpoMinutes=%d want 60", status.Targets.PostgresRPOMinutes)
	}
	if len(status.Tiers) < 2 {
		t.Fatalf("tiers=%d want at least 2", len(status.Tiers))
	}

	now := time.Now().UTC()
	drillBody := map[string]any{
		"drillDate":          now.Format("2006-01-02"),
		"backupTimestamp":    now.Add(-1 * time.Hour).Format(time.RFC3339),
		"restoreStart":       now.Add(-30 * time.Minute).Format(time.RFC3339),
		"restoreEnd":         now.Format(time.RFC3339),
		"rpoAchievedMinutes": 45,
		"rtoAchievedMinutes": 90,
		"pass":               true,
		"smokeTestOutput":    "grade reads, quiz attempts, auth: ok",
	}
	resp2 := backupDo(t, env.srv, http.MethodPost, "/api/v1/internal/ops/restore-drill", drillBody, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("restore-drill: status=%d want 201", resp2.StatusCode)
	}
	var created map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created["id"] == "" {
		t.Fatal("expected drill id")
	}

	resp3 := backupDo(t, env.srv, http.MethodGet, "/api/v1/internal/ops/backup-status", nil, tok)
	defer func() { _ = resp3.Body.Close() }()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("backup-status after drill: status=%d", resp3.StatusCode)
	}
	var status2 struct {
		RestoreDrills []struct {
			ID   string `json:"id"`
			Pass *bool  `json:"pass"`
		} `json:"restoreDrills"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&status2); err != nil {
		t.Fatalf("decode status2: %v", err)
	}
	if len(status2.RestoreDrills) == 0 {
		t.Fatal("expected restore drill in history")
	}
	if status2.RestoreDrills[0].Pass == nil || !*status2.RestoreDrills[0].Pass {
		t.Errorf("pass=%v want true", status2.RestoreDrills[0].Pass)
	}
}

func TestBackupOps_ForbiddenWithoutPermission(t *testing.T) {
	env := setupBackupOps(t)
	ctx := context.Background()
	email := fmt.Sprintf("backup-noadmin-%d@test.example", time.Now().UnixNano())
	ph, _ := auth.HashPassword("Passw0rd!backup2")
	dn := "No Admin"
	u, err := user.InsertUser(ctx, env.pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	tok, err := env.signer.Sign(ctx, u.ID, email, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	resp := backupDo(t, env.srv, http.MethodGet, "/api/v1/internal/ops/backup-status", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d want 403", resp.StatusCode)
	}
}
