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
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

type dataResidencyEnv struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
	email  string
}

func setupDataResidency(t *testing.T) *dataResidencyEnv {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping data residency e2e tests")
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

	signer := auth.NewJWTSigner("datares-e2e-jwt-secret-min32chars-xx")

	email := fmt.Sprintf("datares-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!datares")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "DataResidency Test User"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	// Grant Global Admin role so the user has compliance:data-residency:admin:* permission.
	if err := rbac.AssignUserRoleByName(ctx, pool, userID, "Global Admin"); err != nil {
		t.Fatalf("assign Global Admin: %v", err)
	}

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			DataResidencyEnabled: true,
		},
	}))
	t.Cleanup(srv.Close)

	return &dataResidencyEnv{
		srv:    srv,
		pool:   pool,
		signer: signer,
		userID: userID,
		email:  email,
	}
}

func (e *dataResidencyEnv) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), e.email, "", "", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func drDo(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
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

// TestDataResidency_GetOrgRegion verifies that an org's data residency region is returned (AC-1, FR-7).
func TestDataResidency_GetOrgRegion(t *testing.T) {
	env := setupDataResidency(t)
	tok := env.token(t)
	ctx := context.Background()

	// Create an EU-region org.
	ts := time.Now().UnixNano()
	org, err := organization.Create(ctx, env.pool,
		"EU University DR Test", fmt.Sprintf("eu-university-dr-test-%d", ts),
		nil, nil, "eu-west", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	resp := drDo(t, env.srv, http.MethodGet,
		"/api/v1/internal/compliance/data-residency/org/"+org.ID.String(),
		nil, tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get org region: status=%d want 200", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["dataRegion"] != "eu-west" {
		t.Errorf("dataRegion=%q want eu-west", result["dataRegion"])
	}
	if result["orgId"] != org.ID.String() {
		t.Errorf("orgId=%q want %s", result["orgId"], org.ID)
	}
}

// TestDataResidency_GetOrgRegion_NotFound verifies 404 for unknown org.
func TestDataResidency_GetOrgRegion_NotFound(t *testing.T) {
	env := setupDataResidency(t)
	tok := env.token(t)

	resp := drDo(t, env.srv, http.MethodGet,
		"/api/v1/internal/compliance/data-residency/org/"+uuid.New().String(),
		nil, tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d want 404", resp.StatusCode)
	}
}

// TestDataResidency_AccessLog_Empty verifies the access log returns an empty list when no events exist.
func TestDataResidency_AccessLog_Empty(t *testing.T) {
	env := setupDataResidency(t)
	tok := env.token(t)

	resp := drDo(t, env.srv, http.MethodGet,
		"/api/v1/internal/compliance/data-residency/access-log",
		nil, tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("access-log: status=%d want 200", resp.StatusCode)
	}
	var result struct {
		Entries []any `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Entries may be nil/empty — both are acceptable.
}

// TestDataResidency_RegionImmutable verifies that changing data_region via PATCH returns 422 (FR-4, AC-3).
func TestDataResidency_RegionImmutable(t *testing.T) {
	env := setupDataResidency(t)
	tok := env.token(t)
	ctx := context.Background()

	// Create a US-east org.
	ts := time.Now().UnixNano()
	org, err := organization.Create(ctx, env.pool,
		"US Region Immutable Test", fmt.Sprintf("us-region-immutable-test-%d", ts),
		nil, nil, "us-east", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	// Attempt to change data_region via the admin PATCH endpoint.
	resp := drDo(t, env.srv, http.MethodPatch,
		"/api/v1/admin/orgs/"+org.ID.String(),
		map[string]any{"dataRegion": "eu-west"}, tok)
	defer func() { _ = resp.Body.Close() }()

	// AC-3: must return 422.
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("region change: status=%d want 422", resp.StatusCode)
	}
}

// TestDataResidency_RegionSetAtProvisioning verifies that a new org gets the correct region (FR-1, AC-1).
func TestDataResidency_RegionSetAtProvisioning(t *testing.T) {
	env := setupDataResidency(t)
	tok := env.token(t)
	ctx := context.Background()

	ts := time.Now().UnixNano()
	org, err := organization.Create(ctx, env.pool,
		"CA Region Test", fmt.Sprintf("ca-region-test-%d", ts),
		nil, nil, "ca-central", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	resp := drDo(t, env.srv, http.MethodGet,
		"/api/v1/internal/compliance/data-residency/org/"+org.ID.String(),
		nil, tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want 200", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["dataRegion"] != "ca-central" {
		t.Errorf("dataRegion=%q want ca-central", result["dataRegion"])
	}
}

// TestDataResidency_AccessLog_FilterByOrg verifies filtering the access log by orgId.
func TestDataResidency_AccessLog_FilterByOrg(t *testing.T) {
	env := setupDataResidency(t)
	tok := env.token(t)
	ctx := context.Background()

	ts := time.Now().UnixNano()
	org, err := organization.Create(ctx, env.pool,
		"AU Access Log Test", fmt.Sprintf("au-access-log-test-%d", ts),
		nil, nil, "au-east", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	// Insert a cross-region access log entry directly via DB.
	_, err = env.pool.Exec(ctx, `
INSERT INTO compliance.data_residency_access_log (org_id, org_region, requested_from, event_type, request_path)
VALUES ($1, 'au-east', 'us-east', 'cross_region_access_blocked', '/api/v1/test')
`, org.ID)
	if err != nil {
		t.Fatalf("insert access log: %v", err)
	}

	resp := drDo(t, env.srv, http.MethodGet,
		"/api/v1/internal/compliance/data-residency/access-log?orgId="+org.ID.String(),
		nil, tok)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("access-log filter: status=%d want 200", resp.StatusCode)
	}
	var result struct {
		Entries []struct {
			ID        string `json:"id"`
			OrgID     string `json:"orgId"`
			OrgRegion string `json:"orgRegion"`
			EventType string `json:"eventType"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected at least one entry in access log")
	}
	if result.Entries[0].OrgID != org.ID.String() {
		t.Errorf("orgId=%q want %s", result.Entries[0].OrgID, org.ID)
	}
	if result.Entries[0].OrgRegion != "au-east" {
		t.Errorf("orgRegion=%q want au-east", result.Entries[0].OrgRegion)
	}
	if result.Entries[0].EventType != "cross_region_access_blocked" {
		t.Errorf("eventType=%q want cross_region_access_blocked", result.Entries[0].EventType)
	}
}

// TestDataResidency_GetOrgRegion_Unauthenticated verifies 401 for unauthenticated requests.
func TestDataResidency_GetOrgRegion_Unauthenticated(t *testing.T) {
	env := setupDataResidency(t)

	resp := drDo(t, env.srv, http.MethodGet,
		"/api/v1/internal/compliance/data-residency/org/"+uuid.New().String(),
		nil, "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", resp.StatusCode)
	}
}

// TestDataResidency_GetRegion_RepoHelper verifies the GetRegion repo function returns correct region.
func TestDataResidency_GetRegion_RepoHelper(t *testing.T) {
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

	ts := time.Now().UnixNano()
	org, err := organization.Create(ctx, pool,
		"Region Repo Helper Test", fmt.Sprintf("region-repo-helper-test-%d", ts),
		nil, nil, "eu-west", nil)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	region, err := organization.GetRegion(ctx, pool, org.ID)
	if err != nil {
		t.Fatalf("GetRegion: %v", err)
	}
	if region != "eu-west" {
		t.Errorf("region=%q want eu-west", region)
	}

	// Non-existent org should return empty string, no error.
	region2, err := organization.GetRegion(ctx, pool, uuid.New())
	if err != nil {
		t.Fatalf("GetRegion (missing): %v", err)
	}
	if region2 != "" {
		t.Errorf("region=%q want empty for missing org", region2)
	}
}
