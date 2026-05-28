package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

type securityReportsEnv struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
	email  string
}

func setupSecurityReports(t *testing.T) *securityReportsEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping security reports e2e tests")
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

	signer := auth.NewJWTSigner("security-reports-e2e-jwt-secret-min32chars")

	email := fmt.Sprintf("security-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!security")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "Security Reports Test"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	if err := rbac.AssignUserRoleByName(ctx, pool, userID, "Global Admin"); err != nil {
		t.Fatalf("assign Global Admin: %v", err)
	}

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			SecurityDisclosureModuleEnabled: true,
		},
	}))
	t.Cleanup(srv.Close)

	return &securityReportsEnv{srv: srv, pool: pool, signer: signer, userID: userID, email: email}
}

func (e *securityReportsEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), e.email, "", "", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func securityReportsDo(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
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
		t.Fatalf("do: %v", err)
	}
	return resp
}

func TestSecurityReports_TrustPolicy_Public(t *testing.T) {
	env := setupSecurityReports(t)
	resp := securityReportsDo(t, env.srv, http.MethodGet, "/api/v1/trust/security", nil, "")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("trust/security: status=%d want 200", resp.StatusCode)
	}
	var policy struct {
		ContactEmail              string `json:"contactEmail"`
		CoordinatedDisclosureDays int    `json:"coordinatedDisclosureDays"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if policy.ContactEmail != "security@lextures.io" {
		t.Errorf("contactEmail=%q", policy.ContactEmail)
	}
	if policy.CoordinatedDisclosureDays != 90 {
		t.Errorf("coordinatedDisclosureDays=%d want 90", policy.CoordinatedDisclosureDays)
	}
}

func TestSecurityReports_CriticalSLA(t *testing.T) {
	env := setupSecurityReports(t)
	tok := env.token(t)

	reportDate := time.Now().UTC().AddDate(0, 0, -3).Format("2006-01-02")
	patchDate := time.Now().UTC().Format("2006-01-02")

	resp := securityReportsDo(t, env.srv, http.MethodPost, "/api/v1/compliance/security-reports", map[string]any{
		"summary":    "E2E critical XSS",
		"severity":   "critical",
		"reportDate": reportDate,
		"cvssScore":  9.8,
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status=%d", resp.StatusCode)
	}
	var created map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id := created["id"]
	if id == "" {
		t.Fatal("empty id")
	}

	resp2 := securityReportsDo(t, env.srv, http.MethodPatch, "/api/v1/compliance/security-reports/"+id, map[string]any{
		"status":    "patched",
		"severity":  "critical",
		"patchDate": patchDate,
	}, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("patch: status=%d", resp2.StatusCode)
	}

	resp3 := securityReportsDo(t, env.srv, http.MethodGet, "/api/v1/compliance/security-reports/"+id, nil, tok)
	defer func() { _ = resp3.Body.Close() }()
	var report struct {
		SLAMet   *bool  `json:"slaMet"`
		Status   string `json:"status"`
		Severity string `json:"severity"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&report); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if report.Status != "patched" {
		t.Errorf("status=%q", report.Status)
	}
	if report.SLAMet == nil || !*report.SLAMet {
		t.Errorf("expected slaMet true for critical patched within 7 days, got %v", report.SLAMet)
	}
}

func TestSecurityReports_ExportCSV(t *testing.T) {
	env := setupSecurityReports(t)
	tok := env.token(t)

	resp := securityReportsDo(t, env.srv, http.MethodGet, "/api/v1/compliance/security-reports/export", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export: status=%d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read export body: %v", err)
	}
	if !strings.Contains(string(body), "severity") {
		t.Error("CSV missing severity header")
	}
}
