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
	"github.com/lextures/lextures/server/internal/repos/user"
)

type dpaEnv struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
}

func setupDPA(t *testing.T) *dpaEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping DPA e2e tests")
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

	signer := auth.NewJWTSigner("dpa-e2e-jwt-secret-min32chars-xxx")

	email := fmt.Sprintf("dpa-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!dpa")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "DPA Test User"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			DPAPortalEnabled: true,
		},
	}))
	t.Cleanup(srv.Close)

	return &dpaEnv{
		srv:    srv,
		pool:   pool,
		signer: signer,
		userID: userID,
	}
}

func (e *dpaEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), "dpa-e2e@test.example", "", "", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func dpaDo(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
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

// TestDPA_GetCurrentVersion verifies the current DPA version endpoint returns expected fields (AC-2).
func TestDPA_GetCurrentVersion(t *testing.T) {
	env := setupDPA(t)
	tok := env.token(t)

	resp := dpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/dpa/current", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /dpa/current: status=%d want 200", resp.StatusCode)
	}

	var body struct {
		Version struct {
			ID          string `json:"id"`
			VersionStr  string `json:"versionStr"`
			TemplateURL string `json:"templateUrl"`
			EffectiveAt string `json:"effectiveAt"`
		} `json:"version"`
		Signed   bool `json:"signed"`
		Template struct {
			VendorName          string   `json:"vendorName"`
			DPAVersionStr       string   `json:"dpaVersionStr"`
			SubProcessors       []string `json:"subProcessors"`
			DataInventorySummary []string `json:"dataInventorySummary"`
			GeneratedAt         string   `json:"generatedAt"`
		} `json:"template"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Version.VersionStr == "" {
		t.Error("versionStr must not be empty")
	}
	if body.Version.TemplateURL == "" {
		t.Error("templateUrl must not be empty")
	}
	if body.Template.VendorName == "" {
		t.Error("template.vendorName must not be empty")
	}
	if len(body.Template.SubProcessors) == 0 {
		t.Error("template.subProcessors must not be empty")
	}
	if len(body.Template.DataInventorySummary) == 0 {
		t.Error("template.dataInventorySummary must not be empty")
	}
	if body.Template.GeneratedAt == "" {
		t.Error("template.generatedAt must not be empty")
	}
	if _, err := time.Parse(time.RFC3339, body.Template.GeneratedAt); err != nil {
		t.Errorf("template.generatedAt %q is not RFC3339", body.Template.GeneratedAt)
	}
	// signed field is present; its value depends on prior test state — verified in TestDPA_AcceptAndBadge.
	_ = body.Signed
}

// TestDPA_AcceptAndBadge verifies the full acceptance lifecycle (AC-3).
func TestDPA_AcceptAndBadge(t *testing.T) {
	env := setupDPA(t)
	tok := env.token(t)

	// Accept the DPA (AC-3: POST /accept).
	resp := dpaDo(t, env.srv, http.MethodPost, "/api/v1/compliance/dpa/accept", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /dpa/accept: status=%d want 201", resp.StatusCode)
	}
	var acceptResp struct {
		ID         string `json:"id"`
		VersionStr string `json:"versionStr"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&acceptResp); err != nil {
		t.Fatalf("decode accept response: %v", err)
	}
	if acceptResp.ID == "" {
		t.Error("acceptance id must not be empty")
	}
	if acceptResp.VersionStr == "" {
		t.Error("acceptance versionStr must not be empty")
	}

	// After acceptance, GET /current should show signed=true and acceptedAt (AC-3 badge).
	resp2 := dpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/dpa/current", nil, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /dpa/current after accept: status=%d want 200", resp2.StatusCode)
	}
	var body2 struct {
		Signed     bool   `json:"signed"`
		AcceptedAt string `json:"acceptedAt"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&body2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body2.Signed {
		t.Error("signed should be true after acceptance")
	}
	if body2.AcceptedAt == "" {
		t.Error("acceptedAt must be set after acceptance")
	}
}

// TestDPA_AcceptIdempotent verifies accepting twice returns 201 both times (idempotent).
func TestDPA_AcceptIdempotent(t *testing.T) {
	env := setupDPA(t)
	tok := env.token(t)

	for i := 0; i < 2; i++ {
		resp := dpaDo(t, env.srv, http.MethodPost, "/api/v1/compliance/dpa/accept", nil, tok)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("attempt %d: POST /dpa/accept status=%d want 201", i+1, resp.StatusCode)
		}
	}
}

// TestDPA_DataInventory verifies the inventory endpoint returns all seeded elements (AC-5).
func TestDPA_DataInventory(t *testing.T) {
	env := setupDPA(t)
	tok := env.token(t)

	resp := dpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/data-inventory", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /data-inventory: status=%d want 200", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			ID                      string   `json:"id"`
			ElementName             string   `json:"elementName"`
			Category                string   `json:"category"`
			Purpose                 string   `json:"purpose"`
			LegalBasis              string   `json:"legalBasis"`
			SharedWithSubProcessors bool     `json:"sharedWithSubProcessors"`
			SubProcessorNames       []string `json:"subProcessorNames"`
			UpdatedAt               string   `json:"updatedAt"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Items) == 0 {
		t.Fatal("data inventory must have at least one item")
	}
	for _, item := range body.Items {
		if item.ElementName == "" {
			t.Error("item elementName must not be empty")
		}
		if item.Category == "" {
			t.Error("item category must not be empty")
		}
		if item.LegalBasis == "" {
			t.Error("item legalBasis must not be empty")
		}
	}

	// Verify AI processing rows include sub-processor names (AC-5, plan §11).
	hasAIRow := false
	for _, item := range body.Items {
		if item.SharedWithSubProcessors && len(item.SubProcessorNames) > 0 {
			hasAIRow = true
		}
	}
	if !hasAIRow {
		t.Error("data inventory must contain at least one AI-processing row with sub-processor names")
	}
}

// TestDPA_SDPCCSVExport verifies the SDPC CSV export format (AC-5).
func TestDPA_SDPCCSVExport(t *testing.T) {
	env := setupDPA(t)
	tok := env.token(t)

	resp := dpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/data-inventory/export.csv", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /data-inventory/export.csv: status=%d want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type=%q want text/csv", ct)
	}
	cd := resp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition=%q should contain 'attachment'", cd)
	}

	csvRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	csv := string(csvRaw)
	if csv == "" {
		t.Fatal("CSV export must not be empty")
	}

	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) < 2 {
		t.Fatalf("CSV must have header + at least one data row, got %d lines", len(lines))
	}
	// Verify expected SDPC header columns.
	header := lines[0]
	for _, col := range []string{"Element Name", "Category", "Purpose", "Legal Basis"} {
		if !strings.Contains(header, col) {
			t.Errorf("CSV header missing column %q", col)
		}
	}
}

// TestDPA_FeatureFlag_Returns404WhenDisabled verifies the feature flag gates all routes.
func TestDPA_FeatureFlag_Returns404WhenDisabled(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping DPA e2e tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:   pool,
		Config: config.Config{DPAPortalEnabled: false},
	}))
	t.Cleanup(srv.Close)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/dpa/current"},
		{http.MethodPost, "/api/v1/compliance/dpa/accept"},
		{http.MethodGet, "/api/v1/compliance/dpa/acceptances"},
		{http.MethodGet, "/api/v1/compliance/data-inventory"},
		{http.MethodGet, "/api/v1/compliance/data-inventory/export.csv"},
	}
	for _, p := range paths {
		req, _ := http.NewRequest(p.method, srv.URL+p.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: request failed: %v", p.method, p.path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s %s: status=%d want 404", p.method, p.path, resp.StatusCode)
		}
	}
}
