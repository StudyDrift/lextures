package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/auth/hibp"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	lprepo "github.com/lextures/lextures/server/internal/repos/learnerprofile"
	learnerprofilederivers "github.com/lextures/lextures/server/internal/service/learnerprofile/derivers"
	learnerprofileservice "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

func TestLearnerProfileControls_PauseResetExport_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	cfg := config.Load()
	cfg.LearnerProfileEnabled = true
	cfg.AdminAuditLogEnabled = true
	stub := hibp.StubChecker{Result: hibp.Result{BreachFound: false, HIBPAvailable: true}}
	jwtSecret := "01234567890123456789012345678901"
	lpSvc := learnerprofileservice.New(pool, learnerprofilederivers.StudyRhythmDeriver{Pool: pool})
	d := Deps{
		Pool:                  pool,
		JWTSigner:             auth.NewJWTSignerWithPool(jwtSecret, pool),
		Config:                cfg,
		PasswordChecker:       stub,
		LearnerProfileService: lpSvc,
	}
	h := NewHandler(d)

	email := "lp-controls-" + time.Now().Format("20060102150405.000") + "@e.invalid"
	password := "J7q#xM2pL9vRkW4$hN8zT1cY5bU6nM0aS"
	signupBody, _ := json.Marshal(map[string]any{
		"email":        email,
		"password":     password,
		"display_name": "LP Controls",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(signupBody))
	req = req.WithContext(ctx)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("signup: %d %s", rr.Code, rr.Body.String())
	}
	var signupResp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&signupResp); err != nil {
		t.Fatal(err)
	}
	token, _ := signupResp["access_token"].(string)
	authUser, err := d.JWTSigner.Verify(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.MustParse(authUser.UserID)

	now := time.Now().UTC()
	for i := 0; i < 12; i++ {
		at := now.Add(-time.Duration(i) * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `
INSERT INTO analytics.engagement_events (user_id, event_type, occurred_at)
VALUES ($1, 'heartbeat', $2)
`, userID, at); err != nil {
			t.Fatal(err)
		}
	}
	if err := lpSvc.RecomputeIncremental(ctx, userID, "study_rhythm"); err != nil {
		t.Fatal(err)
	}

	pauseReq := httptest.NewRequest(http.MethodPost, "/api/v1/me/learner-profile/pause", nil)
	pauseReq.Header.Set("Authorization", "Bearer "+token)
	pauseW := httptest.NewRecorder()
	h.ServeHTTP(pauseW, pauseReq)
	if pauseW.Code != http.StatusOK {
		t.Fatalf("pause: %d %s", pauseW.Code, pauseW.Body.String())
	}
	var pauseResp map[string]string
	_ = json.NewDecoder(pauseW.Body).Decode(&pauseResp)
	if pauseResp["status"] != "paused" {
		t.Fatalf("pause status: %+v", pauseResp)
	}

	var facetCountBefore int
	if err := pool.QueryRow(ctx, `
SELECT count(*)::int FROM learner.profile_facets f
JOIN learner.profiles p ON p.id = f.profile_id WHERE p.user_id = $1
`, userID).Scan(&facetCountBefore); err != nil {
		t.Fatal(err)
	}
	if err := lpSvc.RecomputeIncremental(ctx, userID, "study_rhythm"); err != nil {
		t.Fatal(err)
	}
	var facetCountAfter int
	if err := pool.QueryRow(ctx, `
SELECT count(*)::int FROM learner.profile_facets f
JOIN learner.profiles p ON p.id = f.profile_id WHERE p.user_id = $1
`, userID).Scan(&facetCountAfter); err != nil {
		t.Fatal(err)
	}
	if facetCountAfter != facetCountBefore {
		t.Fatalf("recompute changed facets while paused: before=%d after=%d", facetCountBefore, facetCountAfter)
	}
	p, _ := lprepo.GetProfileByUserID(ctx, pool, userID)
	if p == nil || p.Status != "paused" {
		t.Fatalf("expected paused profile, got %+v", p)
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/learner-profile/export", nil)
	exportReq.Header.Set("Authorization", "Bearer "+token)
	exportW := httptest.NewRecorder()
	h.ServeHTTP(exportW, exportReq)
	if exportW.Code != http.StatusOK {
		t.Fatalf("export: %d %s", exportW.Code, exportW.Body.String())
	}
	var exportDoc map[string]any
	if err := json.NewDecoder(exportW.Body).Decode(&exportDoc); err != nil {
		t.Fatal(err)
	}
	if exportDoc["exportKind"] != "learner-profile" {
		t.Fatalf("export kind: %+v", exportDoc["exportKind"])
	}
	facets, _ := exportDoc["facets"].([]any)
	if len(facets) == 0 {
		t.Fatalf("export facets empty: %+v", exportDoc)
	}
	disclosure, _ := exportDoc["disclosure"].(map[string]any)
	if disclosure["art22Posture"] == nil {
		t.Fatalf("missing art22 disclosure: %+v", disclosure)
	}

	resetReq := httptest.NewRequest(http.MethodPost, "/api/v1/me/learner-profile/reset", nil)
	resetReq.Header.Set("Authorization", "Bearer "+token)
	resetW := httptest.NewRecorder()
	h.ServeHTTP(resetW, resetReq)
	if resetW.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", resetW.Code, resetW.Body.String())
	}
	n, err := lprepo.CountProfileRows(ctx, pool, userID)
	if err != nil {
		t.Fatal(err)
	}
	if n > 0 {
		t.Fatalf("expected no learner rows after reset, got %d", n)
	}

	profileReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/learner-profile", nil)
	profileReq.Header.Set("Authorization", "Bearer "+token)
	profileW := httptest.NewRecorder()
	h.ServeHTTP(profileW, profileReq)
	if profileW.Code != http.StatusOK {
		t.Fatalf("profile after reset: %d", profileW.Code)
	}
	var profileResp map[string]any
	_ = json.NewDecoder(profileW.Body).Decode(&profileResp)
	profile, _ := profileResp["profile"].(map[string]any)
	if profile["status"] != "insufficient_data" {
		t.Fatalf("expected insufficient_data after reset, got %+v", profile)
	}

	var auditCount int
	if err := pool.QueryRow(ctx, `
SELECT count(*)::int FROM compliance.admin_audit_log
WHERE actor_id = $1 AND event_type = 'learner_profile_control'
`, userID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount < 3 {
		t.Fatalf("expected audit entries for pause/export/reset, got %d", auditCount)
	}
}