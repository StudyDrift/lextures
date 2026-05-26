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
	"github.com/lextures/lextures/server/internal/repos/user"
)

type gdprEnv struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
}

func setupGDPR(t *testing.T) *gdprEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping GDPR e2e tests")
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

	signer := auth.NewJWTSigner("gdpr-e2e-jwt-secret-min32chars-xx")

	email := fmt.Sprintf("gdpr-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!gdpr")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "GDPR Test User"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			GDPRModuleEnabled: true,
		},
	}))
	t.Cleanup(srv.Close)

	return &gdprEnv{
		srv:    srv,
		pool:   pool,
		signer: signer,
		userID: userID,
	}
}

func (e *gdprEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), "gdpr-e2e@test.example", "", "", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func gdprDo(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
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

// TestGDPR_ConsentGrantAndWithdraw verifies the full consent lifecycle (AC-3 related).
func TestGDPR_ConsentGrantAndWithdraw(t *testing.T) {
	env := setupGDPR(t)
	tok := env.token(t)

	// Grant consent for ai_processing.
	resp := gdprDo(t, env.srv, http.MethodPost, "/api/v1/compliance/gdpr/consents",
		map[string]string{
			"purpose":        "ai_processing",
			"lawfulBasis":    "consent",
			"consentVersion": "1.0",
		}, tok)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("grant consent: status=%d want 201", resp.StatusCode)
	}
	var grantResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&grantResp); err != nil {
		t.Fatalf("decode grant response: %v", err)
	}
	consentID, ok := grantResp["id"]
	if !ok || consentID == "" {
		t.Fatal("grant response missing id")
	}

	// List consents — should have one active entry.
	resp2 := gdprDo(t, env.srv, http.MethodGet, "/api/v1/compliance/gdpr/consents", nil, tok)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("list consents: status=%d want 200", resp2.StatusCode)
	}
	var listResp struct {
		Consents []struct {
			ID          string  `json:"id"`
			Purpose     string  `json:"purpose"`
			WithdrawnAt *string `json:"withdrawnAt"`
		} `json:"consents"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Consents) == 0 {
		t.Fatal("expected at least one consent")
	}
	found := false
	for _, c := range listResp.Consents {
		if c.ID == consentID {
			found = true
			if c.Purpose != "ai_processing" {
				t.Errorf("consent purpose=%q want ai_processing", c.Purpose)
			}
			if c.WithdrawnAt != nil {
				t.Error("newly-granted consent should not have withdrawnAt")
			}
		}
	}
	if !found {
		t.Errorf("consent %s not found in list", consentID)
	}

	// Withdraw the consent (AC-3).
	resp3 := gdprDo(t, env.srv, http.MethodDelete, "/api/v1/compliance/gdpr/consents/"+consentID, nil, tok)
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("withdraw consent: status=%d want 200", resp3.StatusCode)
	}
	var delResp map[string]bool
	if err := json.NewDecoder(resp3.Body).Decode(&delResp); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if !delResp["ok"] {
		t.Error("withdraw consent: ok should be true")
	}

	// List again — consent should now have withdrawnAt set.
	resp4 := gdprDo(t, env.srv, http.MethodGet, "/api/v1/compliance/gdpr/consents", nil, tok)
	defer resp4.Body.Close()
	var listResp2 struct {
		Consents []struct {
			ID          string  `json:"id"`
			WithdrawnAt *string `json:"withdrawnAt"`
		} `json:"consents"`
	}
	if err := json.NewDecoder(resp4.Body).Decode(&listResp2); err != nil {
		t.Fatalf("decode second list response: %v", err)
	}
	for _, c := range listResp2.Consents {
		if c.ID == consentID && c.WithdrawnAt == nil {
			t.Error("withdrawn consent should have withdrawnAt set")
		}
	}
}

// TestGDPR_SubmitDSAR verifies submitting a DSAR creates a row (AC-1 related).
func TestGDPR_SubmitDSAR(t *testing.T) {
	env := setupGDPR(t)
	tok := env.token(t)

	// Submit an access request.
	resp := gdprDo(t, env.srv, http.MethodPost, "/api/v1/compliance/gdpr/dsar",
		map[string]string{"requestType": "access"}, tok)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("submit DSAR: status=%d want 201", resp.StatusCode)
	}
	var createResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	dsarID, ok := createResp["id"]
	if !ok || dsarID == "" {
		t.Fatal("submit DSAR response missing id")
	}

	// List own requests — should contain the new one.
	resp2 := gdprDo(t, env.srv, http.MethodGet, "/api/v1/compliance/gdpr/dsar", nil, tok)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("list DSAR: status=%d want 200", resp2.StatusCode)
	}
	var listResp struct {
		Requests []struct {
			ID          string `json:"id"`
			RequestType string `json:"requestType"`
			Status      string `json:"status"`
			DueAt       string `json:"dueAt"`
		} `json:"requests"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	found := false
	for _, r := range listResp.Requests {
		if r.ID == dsarID {
			found = true
			if r.RequestType != "access" {
				t.Errorf("requestType=%q want access", r.RequestType)
			}
			if r.Status != "pending" {
				t.Errorf("status=%q want pending", r.Status)
			}
			// Verify due_at is ~30 days from now.
			due, err := time.Parse(time.RFC3339, r.DueAt)
			if err != nil {
				t.Errorf("dueAt %q is not RFC3339: %v", r.DueAt, err)
			} else {
				diff := due.Sub(time.Now().UTC())
				if diff < 29*24*time.Hour || diff > 31*24*time.Hour {
					t.Errorf("dueAt should be ~30 days from now, got %v", diff)
				}
			}
		}
	}
	if !found {
		t.Errorf("DSAR %s not found in list", dsarID)
	}

	// Duplicate submission should return 409.
	resp3 := gdprDo(t, env.srv, http.MethodPost, "/api/v1/compliance/gdpr/dsar",
		map[string]string{"requestType": "access"}, tok)
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusConflict {
		t.Errorf("duplicate DSAR: status=%d want 409", resp3.StatusCode)
	}
}

// TestGDPR_FeatureFlag_Returns404WhenDisabled verifies the feature flag gates all routes.
func TestGDPR_FeatureFlag_Returns404WhenDisabled(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping GDPR e2e tests")
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
		Config: config.Config{GDPRModuleEnabled: false},
	}))
	t.Cleanup(srv.Close)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/compliance/gdpr/dsar"},
		{http.MethodGet, "/api/v1/compliance/gdpr/dsar"},
		{http.MethodGet, "/api/v1/compliance/gdpr/consents"},
		{http.MethodGet, "/api/v1/compliance/gdpr/ropa"},
		{http.MethodGet, "/api/v1/compliance/gdpr/dpa-template"},
	}
	for _, p := range paths {
		req, _ := http.NewRequest(p.method, srv.URL+p.path, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: request failed: %v", p.method, p.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s %s: status=%d want 404", p.method, p.path, resp.StatusCode)
		}
	}
}

// TestGDPR_ConsentValidation verifies request validation rejects bad inputs.
func TestGDPR_ConsentValidation(t *testing.T) {
	env := setupGDPR(t)
	tok := env.token(t)

	cases := []struct {
		name   string
		body   map[string]string
		wantStatus int
	}{
		{
			name:       "missing purpose",
			body:       map[string]string{"lawfulBasis": "consent"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid lawfulBasis",
			body:       map[string]string{"purpose": "ai_processing", "lawfulBasis": "invalid"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid",
			body:       map[string]string{"purpose": "analytics", "lawfulBasis": "legitimate_interests"},
			wantStatus: http.StatusCreated,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := gdprDo(t, env.srv, http.MethodPost, "/api/v1/compliance/gdpr/consents", tc.body, tok)
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status=%d want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

// TestGDPR_DSARValidation verifies invalid requestType is rejected.
func TestGDPR_DSARValidation(t *testing.T) {
	env := setupGDPR(t)
	tok := env.token(t)

	resp := gdprDo(t, env.srv, http.MethodPost, "/api/v1/compliance/gdpr/dsar",
		map[string]string{"requestType": "not-a-type"}, tok)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid requestType: status=%d want 400", resp.StatusCode)
	}
}

// TestGDPR_DPATemplate verifies the DPA template endpoint returns a populated response.
func TestGDPR_DPATemplate(t *testing.T) {
	env := setupGDPR(t)
	tok := env.token(t)

	resp := gdprDo(t, env.srv, http.MethodGet, "/api/v1/compliance/gdpr/dpa-template", nil, tok)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dpa-template: status=%d want 200", resp.StatusCode)
	}
	var tpl struct {
		ProcessorName      string   `json:"processorName"`
		SubProcessors      []string `json:"subProcessors"`
		ProcessingPurposes []string `json:"processingPurposes"`
		GeneratedAt        string   `json:"generatedAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tpl); err != nil {
		t.Fatalf("decode DPA template: %v", err)
	}
	if tpl.ProcessorName == "" {
		t.Error("processorName must not be empty")
	}
	if len(tpl.SubProcessors) == 0 {
		t.Error("subProcessors must not be empty")
	}
	if len(tpl.ProcessingPurposes) == 0 {
		t.Error("processingPurposes must not be empty")
	}
	if tpl.GeneratedAt == "" {
		t.Error("generatedAt must not be empty")
	}
}
