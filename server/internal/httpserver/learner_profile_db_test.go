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
	learnerprofilederivers "github.com/lextures/lextures/server/internal/service/learnerprofile/derivers"
	learnerprofileservice "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

func TestLearnerProfile_RecomputeAndRead_Pg(t *testing.T) {
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

	email := "learner-profile-" + time.Now().Format("20060102150405.000") + "@e.invalid"
	password := "J7q#xM2pL9vRkW4$hN8zT1cY5bU6nM0aS"
	signupBody, _ := json.Marshal(map[string]any{
		"email":        email,
		"password":     password,
		"display_name": "LP Tester",
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

	loc, _ := time.LoadLocation("UTC")
	now := time.Now().UTC()
	for day := 0; day < 6; day++ {
		localDay := now.AddDate(0, 0, -(day + 1))
		for i := 0; i < 10; i++ {
			at := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 19, 0, 0, 0, loc).
				Add(time.Duration(i) * time.Minute)
			if _, err := pool.Exec(ctx, `
INSERT INTO analytics.engagement_events (user_id, event_type, occurred_at)
VALUES ($1, 'heartbeat', $2)
`, userID, at); err != nil {
				t.Fatal(err)
			}
		}
	}

	if err := lpSvc.RecomputeIncremental(ctx, userID, "study_rhythm"); err != nil {
		t.Fatal(err)
	}

	profileReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/learner-profile", nil)
	profileReq.Header.Set("Authorization", "Bearer "+token)
	profileW := httptest.NewRecorder()
	h.ServeHTTP(profileW, profileReq)
	if profileW.Code != http.StatusOK {
		t.Fatalf("profile: %d %s", profileW.Code, profileW.Body.String())
	}
	var profileResp map[string]any
	if err := json.NewDecoder(profileW.Body).Decode(&profileResp); err != nil {
		t.Fatal(err)
	}
	profile, _ := profileResp["profile"].(map[string]any)
	facets, _ := profile["facets"].([]any)
	if len(facets) == 0 {
		t.Fatalf("expected facets: %+v", profileResp)
	}
	facet0, _ := facets[0].(map[string]any)
	if facet0["state"] != "ok" {
		t.Fatalf("facet state: %+v", facet0)
	}
	if facet0["facetKey"] != "study_rhythm" {
		t.Fatalf("facetKey: %+v", facet0)
	}

	facetReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/learner-profile/facets/study_rhythm", nil)
	facetReq.Header.Set("Authorization", "Bearer "+token)
	facetW := httptest.NewRecorder()
	h.ServeHTTP(facetW, facetReq)
	if facetW.Code != http.StatusOK {
		t.Fatalf("facet: %d %s", facetW.Code, facetW.Body.String())
	}
	var facetResp map[string]any
	if err := json.NewDecoder(facetW.Body).Decode(&facetResp); err != nil {
		t.Fatal(err)
	}
	insights, _ := facetResp["insights"].([]any)
	if len(insights) == 0 {
		t.Fatalf("expected insights: %+v", facetResp)
	}
	ins0, _ := insights[0].(map[string]any)
	evidence, _ := ins0["evidence"].([]any)
	if len(evidence) == 0 {
		t.Fatalf("expected evidence on insight: %+v", ins0)
	}
	ev0, _ := evidence[0].(map[string]any)
	if ev0["sourceKind"] != "engagement_event" {
		t.Fatalf("evidence: %+v", ev0)
	}
	insKey, _ := ins0["insightKey"].(string)
	if insKey == "" {
		t.Fatalf("insightKey missing: %+v", ins0)
	}

	if err := lpSvc.Erase(ctx, userID); err != nil {
		t.Fatal(err)
	}
	emptyReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/learner-profile", nil)
	emptyReq.Header.Set("Authorization", "Bearer "+token)
	emptyW := httptest.NewRecorder()
	h.ServeHTTP(emptyW, emptyReq)
	if emptyW.Code != http.StatusOK {
		t.Fatalf("profile after erase: %d", emptyW.Code)
	}
	var emptyResp map[string]any
	_ = json.NewDecoder(emptyW.Body).Decode(&emptyResp)
	profile2, _ := emptyResp["profile"].(map[string]any)
	if profile2["status"] != "insufficient_data" {
		t.Fatalf("expected insufficient_data after erase, got %+v", profile2)
	}
}