package httpserver

import (
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
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/atrisk"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/atriskscoring"
)

func TestAtRisk_ScoringAndAlert_Pg(t *testing.T) {
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

	var courseID, enrollmentID uuid.UUID
	err = pool.QueryRow(ctx, `
SELECT c.id, ce.id
FROM course.courses c
INNER JOIN course.course_enrollments ce ON ce.course_id = c.id
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.active
LIMIT 1
`).Scan(&courseID, &enrollmentID)
	if err != nil {
		t.Skip("no student enrollment in DB")
	}

	cfg := config.Load()
	cfg.AtRiskAlertsEnabled = true
	svc := atriskscoring.Service{Pool: pool, Config: cfg}
	in := atriskscoring.SignalInputs{
		MissingPct:   85,
		DaysInactive: 10,
		GradeTrend:   40,
	}
	score, comp := atriskscoring.ComputeWeightedScore(in, atrisk.DefaultConfig(uuid.Nil))
	if score < 60 {
		t.Fatalf("score %v expected >= 60", score)
	}
	day := time.Now().UTC()
	if err := atrisk.UpsertScore(ctx, pool, atrisk.ScoreRow{
		EnrollmentID: enrollmentID,
		ComputedDate: day,
		Score:        score,
		MissingPct:   &in.MissingPct,
		DaysInactive: in.DaysInactive,
		GradeTrend:   &in.GradeTrend,
		TopFactor:    comp.TopFactor,
	}); err != nil {
		t.Fatalf("upsert score: %v", err)
	}
	_, created, err := atrisk.CreateAlert(ctx, pool, enrollmentID, day, score, comp.TopFactor)
	if err != nil || !created {
		t.Fatalf("create alert: created=%v err=%v", created, err)
	}
	n, err := svc.RunForCourse(ctx, courseID, day)
	if err != nil {
		t.Fatalf("run course: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected enrollments scored, got %d", n)
	}
}

func TestAtRisk_List_FeatureDisabled_Unauthenticated401_Nodb(t *testing.T) {
	d := Deps{Config: config.Config{AtRiskAlertsEnabled: false}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/abc/at-risk", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d want 401", rr.Code)
	}
}

func TestAtRisk_AdminRun_Unauthorized_Nodb(t *testing.T) {
	d := Deps{Config: config.Config{AtRiskAlertsEnabled: true}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/at-risk/run", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d want 401, allow=%q body=%s", rr.Code, rr.Header().Get("Allow"), rr.Body.String())
	}
}

func TestAtRisk_Run_Unauthorized_Nodb(t *testing.T) {
	d := Deps{Config: config.Config{AtRiskAlertsEnabled: true}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/courses/abc/at-risk/run", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d want 401, allow=%q body=%s", rr.Code, rr.Header().Get("Allow"), rr.Body.String())
	}
}

func TestAtRisk_List_Unauthorized_Nodb(t *testing.T) {
	d := Deps{Config: config.Config{AtRiskAlertsEnabled: true}}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/abc/at-risk", nil)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestAtRisk_AdminConfigGet_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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

	em := "atrisk-admin-" + time.Now().Format("20060102150405") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	urow, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uidStr := urow.ID
	uid, _ := uuid.Parse(uidStr)
	if err := rbac.AssignUserRoleByName(ctx, pool, uid, "Global Admin"); err != nil {
		t.Fatalf("role: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, uidStr, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	cfg := config.Load()
	cfg.AtRiskAlertsEnabled = true
	d := Deps{Pool: pool, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/at-risk/config", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["threshold"] == nil {
		t.Fatal("expected threshold in config")
	}
}
