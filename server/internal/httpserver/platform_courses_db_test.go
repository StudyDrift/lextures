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
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/course"
	platformcourses "github.com/lextures/lextures/server/internal/repos/platformcourses"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestAdminCoursesStats_OK_Pg(t *testing.T) {
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

	em := "coursestats-" + time.Now().Format("20060102150405") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, _ := uuid.Parse(row.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, uid, "Global Admin"); err != nil {
		t.Fatalf("ga: %v", err)
	}
	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	expected, err := platformcourses.FetchDashboardStats(ctx, pool)
	if err != nil {
		t.Fatalf("repo stats: %v", err)
	}

	h := NewHandler(Deps{Pool: pool, JWTSigner: signer})
	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/courses/stats", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	var out platformcourses.DashboardStats
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out != expected {
		t.Fatalf("API stats %+v != repo stats %+v", out, expected)
	}
}

func TestAdminCoursesAccess_CrossOrgGlobalAdminCanGetCourse_Pg(t *testing.T) {
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

	var otherOrg uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO tenant.organizations (slug, name, status)
VALUES ($1, $2, 'active') RETURNING id
`, "ga-launch-"+time.Now().Format("150405.000"), "Global Admin Launch Org").Scan(&otherOrg); err != nil {
		t.Fatalf("org: %v", err)
	}

	em := "ga-launch-" + time.Now().Format("20060102150405.000") + "@e.com"
	ph, err := auth.HashPassword("longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid := uuid.MustParse(row.ID)
	if err := rbac.AssignUserRoleByName(ctx, pool, uid, "Global Admin"); err != nil {
		t.Fatalf("ga: %v", err)
	}

	courseCode, err := course.RandomCourseCode()
	if err != nil {
		t.Fatal(err)
	}
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id, org_id, published)
VALUES ($1, 'Launch me', $2, $3, true) RETURNING id
`, courseCode, uid, otherOrg).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, em, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer})

	getCourse := func() *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode, nil)
		r = r.WithContext(ctx)
		r.Header.Set("Authorization", "Bearer "+tok)
		h.ServeHTTP(rr, r)
		return rr
	}

	before := getCourse()
	if before.Code != http.StatusNotFound {
		t.Fatalf("GET course before access status = %d, want 404: %s", before.Code, before.Body.String())
	}

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/courses/"+courseID.String()+"/access", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST access status = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	after := getCourse()
	if after.Code != http.StatusOK {
		t.Fatalf("GET course after access status = %d, want 200: %s", after.Code, after.Body.String())
	}
	var payload struct {
		CourseCode            string   `json:"courseCode"`
		ViewerEnrollmentRoles []string `json:"viewerEnrollmentRoles"`
	}
	if err := json.NewDecoder(after.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.CourseCode != courseCode {
		t.Fatalf("courseCode = %q, want %q", payload.CourseCode, courseCode)
	}
	foundTeacher := false
	for _, role := range payload.ViewerEnrollmentRoles {
		if role == "teacher" {
			foundTeacher = true
			break
		}
	}
	if !foundTeacher {
		t.Fatalf("expected teacher in viewerEnrollmentRoles, got %v", payload.ViewerEnrollmentRoles)
	}
}
