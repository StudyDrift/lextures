package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

type piiRedactionEnv struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
	email  string
	logBuf *bytes.Buffer
}

func setupPIIRedactionE2E(t *testing.T) *piiRedactionEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping PII redaction e2e tests")
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

	var logBuf bytes.Buffer
	inner := slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := logging.NewRedactHandler(inner, logging.NewRedactor(logging.RedactorConfig{
		Registry:   logging.NewFieldRegistry(),
		HMACSecret: []byte("pii-e2e-jwt-secret-min32chars-xx"),
	}))
	slog.SetDefault(slog.New(h))
	logging.GlobalRedactionMetrics.Reset()
	t.Cleanup(func() { logging.GlobalRedactionMetrics.Reset() })

	signer := auth.NewJWTSigner("pii-e2e-jwt-secret-min32chars-xx")
	email := fmt.Sprintf("pii-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!pii-redaction-e2e-test-xyz")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "PII E2E User"
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
			AppEnv: "local",
		},
	}))
	t.Cleanup(srv.Close)

	return &piiRedactionEnv{
		srv:    srv,
		pool:   pool,
		signer: signer,
		userID: userID,
		email:  email,
		logBuf: &logBuf,
	}
}

func (e *piiRedactionEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), e.email, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}

func TestPIIRedaction_LoginAndAccessLogs_NoPlaintextEmail(t *testing.T) {
	env := setupPIIRedactionE2E(t)
	env.logBuf.Reset()

	// Emit a structured log line like auth middleware might.
	slog.Info("auth attempt", "email", env.email, "user_id", env.userID.String())

	body := fmt.Sprintf(`{"email":%q,"password":%q}`, env.email, "Passw0rd!pii-redaction-e2e-test-xyz")
	resp, err := http.Post(env.srv.URL+"/api/v1/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("login status=%d body=%s", resp.StatusCode, b)
	}

	out := env.logBuf.String()
	if strings.Contains(out, env.email) {
		t.Fatalf("plaintext email found in logs:\n%s", out)
	}
	if !strings.Contains(out, "[REDACTED:email]") {
		t.Fatalf("expected redacted email marker in logs:\n%s", out)
	}
	if logging.GlobalRedactionMetrics.Snapshot()["email"] == 0 {
		t.Fatal("expected pii_redactions_total email > 0")
	}
}

func TestPIIRedaction_RedactionStatus_GlobalAdmin(t *testing.T) {
	env := setupPIIRedactionE2E(t)
	logging.GlobalRedactionMetrics.Inc("email")

	req, err := http.NewRequest(http.MethodGet, env.srv.URL+"/api/v1/internal/ops/redaction-status", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+env.token(t))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, b)
	}
	var status struct {
		RedactionEnabled   bool              `json:"redactionEnabled"`
		RegisteredFields   []string          `json:"registeredFields"`
		PIIRedactionsTotal map[string]uint64 `json:"piiRedactionsTotal"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !status.RedactionEnabled {
		t.Fatal("redactionEnabled want true")
	}
	if len(status.RegisteredFields) == 0 {
		t.Fatal("registeredFields empty")
	}
	if status.PIIRedactionsTotal["email"] == 0 {
		t.Fatalf("piiRedactionsTotal.email: %#v", status.PIIRedactionsTotal)
	}
}

func TestPIIRedaction_ConfigBlockedInProduction(t *testing.T) {
	c := config.Config{
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           "pii-e2e-jwt-secret-min32chars-xx",
		DisablePIIRedaction: true,
		AppEnv:              "production",
	}
	if c.DatabaseURL == "" {
		c.DatabaseURL = "postgres://a:b@localhost:5432/db"
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected Validate error for production + DISABLE_PII_REDACTION")
	}
}
