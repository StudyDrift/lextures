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
	repo "github.com/lextures/lextures/server/internal/repos/studyreflection"
	"github.com/lextures/lextures/server/internal/service/studyreflection"
)

func TestSelfReflection_StatsAndJournal_Pg(t *testing.T) {
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
	cfg.SelfReflectionEnabled = true
	stub := hibp.StubChecker{Result: hibp.Result{BreachFound: false, HIBPAvailable: true}}
	jwtSecret := "01234567890123456789012345678901"
	d := Deps{
		Pool:            pool,
		JWTSigner:       auth.NewJWTSignerWithPool(jwtSecret, pool),
		Config:          cfg,
		PasswordChecker: stub,
	}
	h := NewHandler(d)

	email := "reflection-test-" + time.Now().Format("20060102150405.000") + "@e.invalid"
	password := "J7q#xM2pL9vRkW4$hN8zT1cY5bU6nM0aS"
	signupBody, _ := json.Marshal(map[string]any{
		"email":        email,
		"password":     password,
		"display_name": "Reflection Tester",
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
	if token == "" {
		t.Fatal("no access_token")
	}
	authUser, err := d.JWTSigner.Verify(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.MustParse(authUser.UserID)

	putBody, _ := json.Marshal(map[string]any{"weeklyHours": 10, "optedIn": true})
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/me/study-goal", bytes.NewReader(putBody))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	h.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("put goal: %d %s", putW.Code, putW.Body.String())
	}

	now := time.Now().UTC()
	weekStart, _ := studyreflection.WeekBounds(now)
	_, err = pool.Exec(ctx, `
INSERT INTO analytics.engagement_events (user_id, event_type, occurred_at)
SELECT $1, 'heartbeat', $2 + (n || ' minutes')::interval
FROM generate_series(0, 5) AS n
`, userID, weekStart)
	if err != nil {
		t.Fatal(err)
	}

	statsReq := httptest.NewRequest(http.MethodGet, "/api/v1/me/study-stats", nil)
	statsReq.Header.Set("Authorization", "Bearer "+token)
	statsW := httptest.NewRecorder()
	h.ServeHTTP(statsW, statsReq)
	if statsW.Code != http.StatusOK {
		t.Fatalf("stats: %d %s", statsW.Code, statsW.Body.String())
	}
	var stats studyreflection.Stats
	if err := json.NewDecoder(statsW.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if !stats.OptedIn || stats.TimeOnTaskSecondsWeek < 150 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	note := "Confused about recursion"
	postBody, _ := json.Marshal(map[string]string{"entryText": note})
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/me/reflection-journal", bytes.NewReader(postBody))
	postReq.Header.Set("Authorization", "Bearer "+token)
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()
	h.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusCreated {
		t.Fatalf("post journal: %d %s", postW.Code, postW.Body.String())
	}

	entries, err := repo.ListJournal(ctx, pool, userID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.EntryText == note {
			found = true
		}
	}
	if !found {
		t.Fatal("journal entry not stored")
	}
}
