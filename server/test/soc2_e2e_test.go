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
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

type soc2Env struct {
	srv    *httptest.Server
	pool   *pgxpool.Pool
	signer *auth.JWTSigner
	userID uuid.UUID
	email  string
}

func setupSOC2(t *testing.T) *soc2Env {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping SOC 2 e2e tests")
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

	signer := auth.NewJWTSigner("soc2-e2e-jwt-secret-min32chars-xx")

	email := fmt.Sprintf("soc2-e2e-%d@test.example", time.Now().UnixNano())
	ph, err := auth.HashPassword("Passw0rd!soc2")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dn := "SOC2 Test User"
	u, err := user.InsertUser(ctx, pool, email, ph, &dn)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID := uuid.MustParse(u.ID)

	// Grant Global Admin role so the user has compliance:soc2:admin:* permission.
	if err := rbac.AssignUserRoleByName(ctx, pool, userID, "Global Admin"); err != nil {
		t.Fatalf("assign Global Admin: %v", err)
	}

	srv := httptest.NewServer(httpserver.NewHandler(httpserver.Deps{
		Pool:      pool,
		JWTSigner: signer,
		Config: config.Config{
			SOC2ModuleEnabled: true,
		},
	}))
	t.Cleanup(srv.Close)

	return &soc2Env{
		srv:    srv,
		pool:   pool,
		signer: signer,
		userID: userID,
		email:  email,
	}
}

func (e *soc2Env) token(t *testing.T) string {
	t.Helper()
	tok, err := e.signer.Sign(context.Background(), e.userID.String(), e.email, "", "", nil)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func soc2Do(t *testing.T, srv *httptest.Server, method, path string, body any, token string) *http.Response {
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

// TestSOC2_EvidenceSummary verifies the evidence dashboard endpoint (AC-2).
func TestSOC2_EvidenceSummary(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	resp := soc2Do(t, env.srv, http.MethodGet, "/api/v1/internal/compliance/soc2/evidence-summary", nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("evidence-summary: status=%d want 200", resp.StatusCode)
	}
	var result struct {
		OpenIncidents  int `json:"openIncidents"`
		RecentReviews  int `json:"recentReviews"`
		VendorsTotal   int `json:"vendorsTotal"`
		VendorsOverdue int `json:"vendorsOverdue"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode evidence-summary: %v", err)
	}
	// Counts must be non-negative.
	if result.OpenIncidents < 0 {
		t.Error("openIncidents must be non-negative")
	}
	if result.RecentReviews < 0 {
		t.Error("recentReviews must be non-negative")
	}
}

// TestSOC2_AccessReviewCRUD verifies creating and listing access reviews (FR-2, AC-2).
func TestSOC2_AccessReviewCRUD(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	nextDue := time.Now().UTC().Add(90 * 24 * time.Hour).Format(time.RFC3339)
	findings := `{"noted":"no excessive privileges found"}`

	// Create a privileged access review.
	resp := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/access-reviews", map[string]any{
		"reviewType":    "privileged",
		"findings":      findings,
		"nextReviewDue": nextDue,
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create review: status=%d want 201", resp.StatusCode)
	}
	var createResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	reviewID := createResp["id"]
	if reviewID == "" {
		t.Fatal("create access review returned empty id")
	}

	// List reviews — should contain the new one.
	resp2 := soc2Do(t, env.srv, http.MethodGet, "/api/v1/internal/compliance/soc2/access-reviews", nil, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("list reviews: status=%d want 200", resp2.StatusCode)
	}
	var listResp struct {
		Reviews []struct {
			ID         string `json:"id"`
			ReviewType string `json:"reviewType"`
			ReviewedAt string `json:"reviewedAt"`
		} `json:"reviews"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	found := false
	for _, r := range listResp.Reviews {
		if r.ID == reviewID {
			found = true
			if r.ReviewType != "privileged" {
				t.Errorf("reviewType=%q want privileged", r.ReviewType)
			}
		}
	}
	if !found {
		t.Errorf("review %s not found in list", reviewID)
	}
}

// TestSOC2_AccessReview_InvalidType returns 400 for unknown review types.
func TestSOC2_AccessReview_InvalidType(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	resp := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/access-reviews", map[string]any{
		"reviewType": "not-a-type",
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid reviewType: status=%d want 400", resp.StatusCode)
	}
}

// TestSOC2_IncidentLifecycle verifies create → get → update incident (FR-4, AC-3).
func TestSOC2_IncidentLifecycle(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	// Create an incident.
	resp := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/incidents", map[string]any{
		"title":       "Unauthorized access attempt detected",
		"severity":    "P1",
		"tscCriteria": []string{"CC6.1", "CC7.3"},
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create incident: status=%d want 201", resp.StatusCode)
	}
	var createResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	incID := createResp["id"]
	if incID == "" {
		t.Fatal("create incident returned empty id")
	}

	// Get the incident by ID.
	resp2 := soc2Do(t, env.srv, http.MethodGet, "/api/v1/internal/compliance/soc2/incidents/"+incID, nil, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("get incident: status=%d want 200", resp2.StatusCode)
	}
	var incResp struct {
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		Severity string   `json:"severity"`
		Status   string   `json:"status"`
		TSC      []string `json:"tscCriteria"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&incResp); err != nil {
		t.Fatalf("decode incident: %v", err)
	}
	if incResp.ID != incID {
		t.Errorf("id=%q want %q", incResp.ID, incID)
	}
	if incResp.Status != "open" {
		t.Errorf("status=%q want open", incResp.Status)
	}
	if incResp.Severity != "P1" {
		t.Errorf("severity=%q want P1", incResp.Severity)
	}

	// List incidents and find ours.
	resp3 := soc2Do(t, env.srv, http.MethodGet, "/api/v1/internal/compliance/soc2/incidents", nil, tok)
	defer func() { _ = resp3.Body.Close() }()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("list incidents: status=%d want 200", resp3.StatusCode)
	}
	var listResp struct {
		Incidents []struct {
			ID string `json:"id"`
		} `json:"incidents"`
	}
	if err := json.NewDecoder(resp3.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	found := false
	for _, i := range listResp.Incidents {
		if i.ID == incID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("incident %s not found in list", incID)
	}

	// Transition to contained.
	resp4 := soc2Do(t, env.srv, http.MethodPatch, "/api/v1/internal/compliance/soc2/incidents/"+incID, map[string]any{
		"status": "contained",
	}, tok)
	defer func() { _ = resp4.Body.Close() }()
	if resp4.StatusCode != http.StatusOK {
		t.Fatalf("update incident contained: status=%d want 200", resp4.StatusCode)
	}

	// Resolve with post-mortem (AC-3: must be completed within 5 business days).
	postMortemURL := "https://docs.example.com/incident/" + incID
	resp5 := soc2Do(t, env.srv, http.MethodPatch, "/api/v1/internal/compliance/soc2/incidents/"+incID, map[string]any{
		"status":        "resolved",
		"postMortemUrl": postMortemURL,
	}, tok)
	defer func() { _ = resp5.Body.Close() }()
	if resp5.StatusCode != http.StatusOK {
		t.Fatalf("resolve incident: status=%d want 200", resp5.StatusCode)
	}

	// Verify resolved state.
	resp6 := soc2Do(t, env.srv, http.MethodGet, "/api/v1/internal/compliance/soc2/incidents/"+incID, nil, tok)
	defer func() { _ = resp6.Body.Close() }()
	var resolved struct {
		Status        string `json:"status"`
		ResolvedAt    string `json:"resolvedAt"`
		PostMortemURL string `json:"postMortemUrl"`
	}
	if err := json.NewDecoder(resp6.Body).Decode(&resolved); err != nil {
		t.Fatalf("decode resolved: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Errorf("status=%q want resolved", resolved.Status)
	}
	if resolved.ResolvedAt == "" {
		t.Error("resolvedAt must be set after resolution")
	}
	if resolved.PostMortemURL != postMortemURL {
		t.Errorf("postMortemUrl=%q want %q", resolved.PostMortemURL, postMortemURL)
	}
}

// TestSOC2_Incident_MissingTitle returns 400.
func TestSOC2_Incident_MissingTitle(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	resp := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/incidents", map[string]any{
		"title":    "",
		"severity": "P2",
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing title: status=%d want 400", resp.StatusCode)
	}
}

// TestSOC2_Incident_InvalidSeverity returns 400.
func TestSOC2_Incident_InvalidSeverity(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	resp := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/incidents", map[string]any{
		"title":    "Test incident",
		"severity": "CRITICAL",
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid severity: status=%d want 400", resp.StatusCode)
	}
}

// TestSOC2_Incident_NotFound returns 404 for missing UUID.
func TestSOC2_Incident_NotFound(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	resp := soc2Do(t, env.srv, http.MethodGet, "/api/v1/internal/compliance/soc2/incidents/"+uuid.New().String(), nil, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("not found: status=%d want 404", resp.StatusCode)
	}
}

// TestSOC2_VendorRiskRegister verifies vendor upsert and listing (FR-6, AC-6).
func TestSOC2_VendorRiskRegister(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	reportDate := time.Now().UTC().AddDate(0, -3, 0).Format("2006-01-02")
	nextReviewDue := time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02")
	reportURL := "https://trust.openrouter.ai/soc2"
	notes := "AI model routing sub-processor"

	// Add OpenRouter as a critical vendor.
	resp := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/vendor-risk", map[string]any{
		"vendorName":    "OpenRouter",
		"riskTier":      "critical",
		"soc2ReportUrl": reportURL,
		"reportDate":    reportDate,
		"nextReviewDue": nextReviewDue,
		"notes":         notes,
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add vendor: status=%d want 201", resp.StatusCode)
	}
	var createResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create vendor: %v", err)
	}
	if createResp["id"] == "" {
		t.Fatal("add vendor returned empty id")
	}

	// List vendors — OpenRouter must be present (AC-6).
	resp2 := soc2Do(t, env.srv, http.MethodGet, "/api/v1/internal/compliance/soc2/vendor-risk", nil, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("list vendors: status=%d want 200", resp2.StatusCode)
	}
	var listResp struct {
		Vendors []struct {
			VendorName    string `json:"vendorName"`
			RiskTier      string `json:"riskTier"`
			SOC2ReportURL string `json:"soc2ReportUrl"`
		} `json:"vendors"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list vendors: %v", err)
	}
	found := false
	for _, v := range listResp.Vendors {
		if v.VendorName == "OpenRouter" {
			found = true
			if v.RiskTier != "critical" {
				t.Errorf("riskTier=%q want critical", v.RiskTier)
			}
			if v.SOC2ReportURL != reportURL {
				t.Errorf("soc2ReportUrl=%q want %q", v.SOC2ReportURL, reportURL)
			}
		}
	}
	if !found {
		t.Error("OpenRouter not found in vendor risk register")
	}

	// Upsert (update) — change risk tier.
	resp3 := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/vendor-risk", map[string]any{
		"vendorName": "OpenRouter",
		"riskTier":   "high",
	}, tok)
	defer func() { _ = resp3.Body.Close() }()
	if resp3.StatusCode != http.StatusCreated {
		t.Fatalf("upsert vendor: status=%d want 201", resp3.StatusCode)
	}
}

// TestSOC2_Vendor_InvalidRiskTier returns 400.
func TestSOC2_Vendor_InvalidRiskTier(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	resp := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/vendor-risk", map[string]any{
		"vendorName": "TestVendor",
		"riskTier":   "extreme",
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid riskTier: status=%d want 400", resp.StatusCode)
	}
}

// TestSOC2_FeatureFlag_Returns404WhenDisabled verifies the feature flag gates all routes.
func TestSOC2_FeatureFlag_Returns404WhenDisabled(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping SOC 2 e2e tests")
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
		Config: config.Config{SOC2ModuleEnabled: false},
	}))
	t.Cleanup(srv.Close)

	paths := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/internal/compliance/soc2/evidence-summary"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/access-reviews"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/access-reviews"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/incidents"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/incidents"},
		{http.MethodGet, "/api/v1/internal/compliance/soc2/vendor-risk"},
		{http.MethodPost, "/api/v1/internal/compliance/soc2/vendor-risk"},
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

// TestSOC2_ListIncidents_FilterByStatus verifies status query param filtering.
func TestSOC2_ListIncidents_FilterByStatus(t *testing.T) {
	env := setupSOC2(t)
	tok := env.token(t)

	// Create an open incident.
	resp := soc2Do(t, env.srv, http.MethodPost, "/api/v1/internal/compliance/soc2/incidents", map[string]any{
		"title":    "Filter test incident",
		"severity": "P3",
	}, tok)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create incident: status=%d want 201", resp.StatusCode)
	}

	// Filter by open status.
	resp2 := soc2Do(t, env.srv, http.MethodGet, "/api/v1/internal/compliance/soc2/incidents?status=open", nil, tok)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("list open incidents: status=%d want 200", resp2.StatusCode)
	}
	var result struct {
		Incidents []struct {
			Status string `json:"status"`
		} `json:"incidents"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, i := range result.Incidents {
		if i.Status != "open" {
			t.Errorf("filtered result has status=%q, want open", i.Status)
		}
	}
}
