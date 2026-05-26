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
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/instructorinsights"
)

func TestInsights_OK_Pg(t *testing.T) {
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

	// Create instructor user.
	em := "inst-insights-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid := mustUUID(row.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, uid, "Instructor"); err != nil {
		t.Fatalf("role: %v", err)
	}

	// Create a course and enroll the instructor.
	courseCode := "INSIGHTS-" + time.Now().Format("150405000")
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id)
VALUES ($1, 'Insights Test Course', $2)
RETURNING id
`, courseCode, uid).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	var enrollID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, 'teacher')
RETURNING id
`, courseID, uid).Scan(&enrollID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.user_course_grants (user_id, course_id, permission_string)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
`, uid, courseID, "course:"+courseCode+":gradebook:view"); err != nil {
		t.Fatalf("grant: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	cfg := config.Config{InstructorInsightsEnabled: true}
	d := Deps{Pool: pool, JWTSigner: signer, Config: cfg}
	h := NewHandler(d)

	// GET insights — empty course should return valid empty signals.
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/analytics/insights", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET insights: %d %s", rr.Code, rr.Body.String())
	}
	var ins instructorinsights.Insights
	if err := json.NewDecoder(rr.Body).Decode(&ins); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ins.WorkingWell == nil {
		t.Fatal("workingWell must not be nil")
	}
	if ins.NeedsAttention == nil {
		t.Fatal("needsAttention must not be nil")
	}

	// GET cross-section — no sections → empty list.
	rr2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet,
		"/api/v1/courses/"+courseCode+"/analytics/cross-section", nil)
	r2 = r2.WithContext(ctx)
	r2.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr2, r2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("GET cross-section: %d %s", rr2.Code, rr2.Body.String())
	}

	// POST refresh.
	rr3 := httptest.NewRecorder()
	r3 := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+courseCode+"/analytics/insights/refresh", nil)
	r3 = r3.WithContext(ctx)
	r3.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr3, r3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("POST refresh: %d %s", rr3.Code, rr3.Body.String())
	}

	// POST dismiss — use a fake signal key (item ID).
	fakeKey := uuid.New().String()
	body := map[string]string{"signalKey": fakeKey, "reason": "Known issue"}
	bodyBytes, _ := json.Marshal(body)
	rr4 := httptest.NewRecorder()
	r4 := httptest.NewRequest(http.MethodPost,
		"/api/v1/courses/"+courseCode+"/analytics/insights/dismiss",
		bytes.NewReader(bodyBytes))
	r4 = r4.WithContext(ctx)
	r4.Header.Set("Authorization", "Bearer "+tok)
	r4.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr4, r4)
	if rr4.Code != http.StatusOK {
		t.Fatalf("POST dismiss: %d %s", rr4.Code, rr4.Body.String())
	}
	var dismissed map[string]any
	if err := json.NewDecoder(rr4.Body).Decode(&dismissed); err != nil {
		t.Fatal(err)
	}
	if dismissed["dismissed"] != true {
		t.Fatalf("dismissed != true: %v", dismissed)
	}
}

func TestInsights_Forbidden_Student_Pg(t *testing.T) {
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

	em := "student-insights-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, _ := auth.HashPassword("longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, mustUUID(row.ID), "Student"); err != nil {
		t.Fatalf("role: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	_ = course.GetIDByCourseCode // ensure import used
	cfg := config.Config{InstructorInsightsEnabled: true}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: cfg})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/nonexistent/analytics/insights", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	// Should be 403 (course access denied) or 404 (no such course). Not 200.
	if rr.Code == http.StatusOK {
		t.Fatalf("student should not get 200 on insights: got %d", rr.Code)
	}
}

func TestInsights_FeatureGate_Pg(t *testing.T) {
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

	em := "gate-insights-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, _ := auth.HashPassword("longpassword0")
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if err := rbac.AssignUserRoleByName(ctx, pool, mustUUID(row.ID), "Instructor"); err != nil {
		t.Fatalf("role: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	// Feature disabled.
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{InstructorInsightsEnabled: false}})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/cs101/analytics/insights", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 when feature disabled, got %d", rr.Code)
	}
}
