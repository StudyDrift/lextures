package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// Pins the toolkit-converted outcomes list/create contract (TD.7 AC-1, AC-5).
func TestCourseOutcomes_ToolkitPreservesContract_Pg(t *testing.T) {
	if testing.Short() {
		t.Skip("needs DATABASE_URL")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL")
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

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	ph, err := auth.HashPassword("toolkit-outcomes-password-0")
	if err != nil {
		t.Fatal(err)
	}
	ts := time.Now().UTC().Format("20060102150405.000000000")
	inst, err := user.InsertUser(ctx, pool, "out-kit-inst-"+ts+"@e.com", ph, strPtr("Instructor"))
	if err != nil {
		t.Fatal(err)
	}
	stu, err := user.InsertUser(ctx, pool, "out-kit-stu-"+ts+"@e.com", ph, strPtr("Student"))
	if err != nil {
		t.Fatal(err)
	}
	instID := uuid.MustParse(inst.ID)
	stuID := uuid.MustParse(stu.ID)

	courseCode := "C-" + fmt.Sprintf("%06X", time.Now().UnixNano()%0xFFFFFF)
	var courseID uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO course.courses (title, course_code, org_id, created_by_user_id)
VALUES ('Toolkit Outcomes', $2, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1), $1)
RETURNING id, course_code
`, instID, courseCode).Scan(&courseID, &courseCode)
	if err != nil {
		t.Fatalf("course: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct {
		uid  uuid.UUID
		role string
	}{
		{instID, "teacher"},
		{stuID, "student"},
	} {
		if _, err = tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
`, courseID, pair.uid, pair.role); err != nil {
			t.Fatalf("enroll: %v", err)
		}
		if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, pair.uid, courseID, courseCode); err != nil {
			t.Fatalf("grants: %v", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	instTok, err := signer.Sign(ctx, inst.ID, inst.Email, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	stuTok, err := signer.Sign(ctx, stu.ID, stu.Email, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer})

	// Learner is 403 with the same messages as the hand-rolled handlers.
	for _, tc := range []struct {
		method, path, msg string
	}{
		{http.MethodGet, "/api/v1/courses/" + courseCode + "/outcomes", "You do not have permission to view outcomes."},
		{http.MethodPost, "/api/v1/courses/" + courseCode + "/outcomes", "You do not have permission to edit outcomes."},
	} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{"title":"No"}`))
		req.Header.Set("Authorization", "Bearer "+stuTok)
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("learner %s: %d %s", tc.method, rr.Code, rr.Body.String())
		}
		var env apierr.Body
		if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if env.Error.Code != apierr.CodeForbidden || env.Error.Message != tc.msg {
			t.Fatalf("learner %s envelope=%+v want %q", tc.method, env, tc.msg)
		}
	}

	// Staff create + list.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/outcomes", strings.NewReader(`{"title":"CLO toolkit","description":"d"}`))
	req.Header.Set("Authorization", "Bearer "+instTok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rr.Code, rr.Body.String())
	}
	var created courseOutcomeAPI
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Title != "CLO toolkit" || created.ID == "" {
		t.Fatalf("created=%+v", created)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/courses/"+courseCode+"/outcomes", nil)
	req.Header.Set("Authorization", "Bearer "+instTok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rr.Code, rr.Body.String())
	}
	var list courseOutcomesListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range list.Outcomes {
		if o.ID == created.ID {
			found = true
			if o.Title != "CLO toolkit" {
				t.Fatalf("listed=%+v", o)
			}
		}
	}
	if !found {
		t.Fatalf("created outcome missing from list: %+v", list)
	}

	// Invalid JSON still 400 INVALID_INPUT with the original message.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+courseCode+"/outcomes", strings.NewReader(`{`))
	req.Header.Set("Authorization", "Bearer "+instTok)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad json: %d %s", rr.Code, rr.Body.String())
	}
	var env apierr.Body
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != apierr.CodeInvalidInput || env.Error.Message != "Invalid JSON body." {
		t.Fatalf("bad json envelope=%+v", env)
	}
}
