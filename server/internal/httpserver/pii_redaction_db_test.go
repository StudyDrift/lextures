package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	serverdata "github.com/lextures/lextures/server"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestRedactionStatus_GlobalAdmin_Pg(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	signer := auth.NewJWTSigner("pii-db-test-jwt-secret-min32chars-x")
	email := "pii-redact-db-" + time.Now().UTC().Format("20060102150405.000000") + "@test.invalid"
	ph, err := auth.HashPassword("Passw0rd!pii-db-test-secret-long")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "PII DB Test"
	created, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	uid := uuid.MustParse(created.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, uid, "Global Admin"); err != nil {
		t.Fatalf("role: %v", err)
	}
	tok, err := signer.Sign(ctx, uid.String(), email, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	logging.GlobalRedactionMetrics.Reset()
	logging.GlobalRedactionMetrics.Inc("email")

	h := NewHandler(Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config:    config.Config{AppEnv: "local"},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/internal/ops/redaction-status", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var status struct {
		RedactionEnabled   bool              `json:"redactionEnabled"`
		PIIRedactionsTotal map[string]uint64 `json:"piiRedactionsTotal"`
	}
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !status.RedactionEnabled {
		t.Fatal("redactionEnabled want true")
	}
	if status.PIIRedactionsTotal["email"] == 0 {
		t.Fatalf("metrics: %#v", status.PIIRedactionsTotal)
	}
}
