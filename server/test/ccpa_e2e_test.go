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

type ccpaEnv struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
}

func setupCCPA(t *testing.T) *ccpaEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping CCPA e2e tests")
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

	signer := auth.NewJWTSigner("ccpa-e2e-jwt-secret-min32chars-xx")

	email := fmt.Sprintf("ccpa-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!ccpa")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "CCPA Test User"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			CCPAModuleEnabled: true,
		},
	}))
	t.Cleanup(srv.Close)

	return &ccpaEnv{
		srv:    srv,
		pool:   pool,
		signer: signer,
		userID: userID,
	}
}

func (e *ccpaEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), "ccpa-e2e@test.example", "", "", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func ccpaDo(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
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

// TestCCPA_OptOutToggle verifies the Do Not Sell toggle (AC-4).
func TestCCPA_OptOutToggle(t *testing.T) {
	env := setupCCPA(t)
	tok := env.token(t)

	// Initial state: opted in (do_not_sell = false).
	resp := ccpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/ccpa/opt-out", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get opt-out: status=%d want 200", resp.StatusCode)
	}
	var initial struct {
		DoNotSell        bool `json:"doNotSell"`
		LimitSensitivePI bool `json:"limitSensitivePI"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&initial); err != nil {
		t.Fatalf("decode initial: %v", err)
	}
	if initial.DoNotSell {
		t.Error("initial doNotSell should be false")
	}

	// Opt out of sale.
	trueVal := true
	resp2 := ccpaDo(t, env.srv, http.MethodPost, "/api/v1/compliance/ccpa/opt-out",
		map[string]any{"doNotSell": trueVal}, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("opt out: status=%d want 200", resp2.StatusCode)
	}
	var after struct {
		DoNotSell   bool `json:"doNotSell"`
		GpcHonoured bool `json:"gpcHonoured"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&after); err != nil {
		t.Fatalf("decode after opt-out: %v", err)
	}
	if !after.DoNotSell {
		t.Error("doNotSell should be true after opt-out")
	}
	if after.GpcHonoured {
		t.Error("gpcHonoured should be false for explicit opt-out (no Sec-GPC header)")
	}

	// Verify persisted.
	resp3 := ccpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/ccpa/opt-out", nil, tok)
	defer func() { _ = resp3.Body.Close() }()
	var persisted struct {
		DoNotSell bool `json:"doNotSell"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&persisted); err != nil {
		t.Fatalf("decode persisted: %v", err)
	}
	if !persisted.DoNotSell {
		t.Error("doNotSell should remain true after persistence check")
	}

	// Opt back in.
	falseVal := false
	resp4 := ccpaDo(t, env.srv, http.MethodPost, "/api/v1/compliance/ccpa/opt-out",
		map[string]any{"doNotSell": falseVal}, tok)
	defer func() { _ = resp4.Body.Close() }()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("opt back in: status=%d want 200", resp4.StatusCode)
	}
}

// TestCCPA_GPCHeader verifies the Sec-GPC header is processed as an automatic opt-out (AC-1).
func TestCCPA_GPCHeader(t *testing.T) {
	env := setupCCPA(t)
	tok := env.token(t)

	req, err := http.NewRequest(http.MethodPost, env.srv.URL+"/api/v1/compliance/ccpa/opt-out",
		bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Sec-GPC", "1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("gpc request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("gpc opt-out: status=%d want 200", resp.StatusCode)
	}
	var result struct {
		DoNotSell   bool `json:"doNotSell"`
		GpcHonoured bool `json:"gpcHonoured"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode gpc result: %v", err)
	}
	if !result.DoNotSell {
		t.Error("GPC signal should set doNotSell=true")
	}
	if !result.GpcHonoured {
		t.Error("gpcHonoured should be true when Sec-GPC: 1 is present")
	}
}

// TestCCPA_SubmitRequest verifies submitting a rights request (AC-2).
func TestCCPA_SubmitRequest(t *testing.T) {
	env := setupCCPA(t)
	tok := env.token(t)

	// Submit a know_categories request.
	resp := ccpaDo(t, env.srv, http.MethodPost, "/api/v1/compliance/ccpa/requests",
		map[string]string{"requestType": "know_categories"}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("submit request: status=%d want 201", resp.StatusCode)
	}
	var createResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	reqID, ok := createResp["id"]
	if !ok || reqID == "" {
		t.Fatal("submit response missing id")
	}

	// List own requests — should contain the new one.
	resp2 := ccpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/ccpa/requests", nil, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("list requests: status=%d want 200", resp2.StatusCode)
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
		if r.ID == reqID {
			found = true
			if r.RequestType != "know_categories" {
				t.Errorf("requestType=%q want know_categories", r.RequestType)
			}
			if r.Status != "pending" {
				t.Errorf("status=%q want pending", r.Status)
			}
			// Verify due_at is ~45 days from now (CPRA § 1798.130(a)(2)).
			due, err := time.Parse(time.RFC3339, r.DueAt)
			if err != nil {
				t.Errorf("dueAt %q is not RFC3339: %v", r.DueAt, err)
			} else {
				diff := due.Sub(time.Now().UTC())
				if diff < 44*24*time.Hour || diff > 46*24*time.Hour {
					t.Errorf("dueAt should be ~45 days from now, got %v", diff)
				}
			}
		}
	}
	if !found {
		t.Errorf("request %s not found in list", reqID)
	}

	// Duplicate submission should return 409.
	resp3 := ccpaDo(t, env.srv, http.MethodPost, "/api/v1/compliance/ccpa/requests",
		map[string]string{"requestType": "know_categories"}, tok)
	defer func() { _ = resp3.Body.Close() }()
	if resp3.StatusCode != http.StatusConflict {
		t.Errorf("duplicate request: status=%d want 409", resp3.StatusCode)
	}

	// Get the individual request.
	resp4 := ccpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/ccpa/requests/"+reqID, nil, tok)
	defer func() { _ = resp4.Body.Close() }()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("get request: status=%d want 200", resp4.StatusCode)
	}
}

// TestCCPA_FeatureFlag_Returns404WhenDisabled verifies the feature flag gates all routes.
func TestCCPA_FeatureFlag_Returns404WhenDisabled(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping CCPA e2e tests")
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
		Config: config.Config{CCPAModuleEnabled: false},
	}))
	t.Cleanup(srv.Close)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/compliance/ccpa/opt-out"},
		{http.MethodPost, "/api/v1/compliance/ccpa/opt-out"},
		{http.MethodPost, "/api/v1/compliance/ccpa/requests"},
		{http.MethodGet, "/api/v1/compliance/ccpa/requests"},
		{http.MethodGet, "/api/v1/compliance/ccpa/pi-categories"},
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

// TestCCPA_RequestValidation verifies invalid requestType is rejected.
func TestCCPA_RequestValidation(t *testing.T) {
	env := setupCCPA(t)
	tok := env.token(t)

	resp := ccpaDo(t, env.srv, http.MethodPost, "/api/v1/compliance/ccpa/requests",
		map[string]string{"requestType": "not-a-type"}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid requestType: status=%d want 400", resp.StatusCode)
	}
}

// TestCCPA_PICategories verifies the public PI disclosure endpoint (FR-7).
func TestCCPA_PICategories(t *testing.T) {
	env := setupCCPA(t)

	// Public endpoint — no auth required.
	resp := ccpaDo(t, env.srv, http.MethodGet, "/api/v1/compliance/ccpa/pi-categories", nil, "")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pi-categories: status=%d want 200", resp.StatusCode)
	}
	var result struct {
		Categories []struct {
			Category string `json:"category"`
			Purpose  string `json:"purpose"`
		} `json:"categories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode pi-categories: %v", err)
	}
	if len(result.Categories) == 0 {
		t.Error("pi-categories must return at least one category")
	}
	for i, c := range result.Categories {
		if c.Category == "" {
			t.Errorf("categories[%d].category must not be empty", i)
		}
		if c.Purpose == "" {
			t.Errorf("categories[%d].purpose must not be empty", i)
		}
	}
}

// TestCCPA_LimitSensitivePI verifies the limit sensitive PI toggle.
func TestCCPA_LimitSensitivePI(t *testing.T) {
	env := setupCCPA(t)
	tok := env.token(t)

	trueVal := true
	resp := ccpaDo(t, env.srv, http.MethodPost, "/api/v1/compliance/ccpa/opt-out",
		map[string]any{"limitSensitivePI": trueVal}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("limit sensitive PI: status=%d want 200", resp.StatusCode)
	}
	var result struct {
		LimitSensitivePI bool `json:"limitSensitivePI"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !result.LimitSensitivePI {
		t.Error("limitSensitivePI should be true after setting")
	}
}
