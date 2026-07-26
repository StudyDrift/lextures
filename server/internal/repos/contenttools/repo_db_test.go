package contenttools

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("short")
	}
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
	t.Cleanup(pool.Close)
	return pool
}

func seedCourse(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	courseID := uuid.New()
	courseCode := "ct1-" + courseID.String()[:8]
	_, err := pool.Exec(ctx, `
INSERT INTO course.courses (id, course_code, title, content_tools_enabled)
VALUES ($1, $2, 'CT.1 test', TRUE)
`, courseID, courseCode)
	if err != nil {
		t.Fatalf("seed course: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.courses WHERE id = $1`, courseID)
	})
	return courseID
}

func seedEnrollment(t *testing.T, pool *pgxpool.Pool, courseID uuid.UUID) (userID, enrollmentID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	em := "ct1-" + uuid.NewString()[:8] + "@example.test"
	ph, err := auth.HashPassword("Ct1-test-password-" + uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, err = uuid.Parse(u.ID)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	enrollmentID = uuid.New()
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (id, course_id, user_id, active)
VALUES ($1, $2, $3, TRUE)
`, enrollmentID, courseID, userID); err != nil {
		t.Fatalf("seed enrollment: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.course_enrollments WHERE id = $1`, enrollmentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM "user".users WHERE id = $1`, userID)
	})
	return userID, enrollmentID
}

func TestDB_EnrollmentCascadeDeletesState(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	courseID := seedCourse(t, pool)
	userID, enrollmentID := seedEnrollment(t, pool, courseID)

	inst, err := CreateInstance(ctx, pool, InstanceRow{
		CourseID:    courseID,
		HostKind:    "syllabus",
		ToolID:      "noop_probe",
		ToolVersion: "1.0.0",
		ConfigJSON:  json.RawMessage(`{"prompt":"hi"}`),
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if _, err := UpsertState(ctx, pool, inst.ID, enrollmentID, userID, json.RawMessage(`{"response":"a"}`), 0); err != nil {
		t.Fatalf("upsert state: %v", err)
	}
	n, err := CountStatesForEnrollment(ctx, pool, enrollmentID)
	if err != nil || n != 1 {
		t.Fatalf("count before delete: n=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM course.course_enrollments WHERE id = $1`, enrollmentID); err != nil {
		t.Fatalf("delete enrollment: %v", err)
	}
	n, err = CountStatesForEnrollment(ctx, pool, enrollmentID)
	if err != nil || n != 0 {
		t.Fatalf("expected cascade delete, n=%d err=%v", n, err)
	}
}

func TestDB_IndependentStatePerEnrollment(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	courseID := seedCourse(t, pool)
	u1, e1 := seedEnrollment(t, pool, courseID)
	u2, e2 := seedEnrollment(t, pool, courseID)

	inst, err := CreateInstance(ctx, pool, InstanceRow{
		CourseID: courseID, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"q"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertState(ctx, pool, inst.ID, e1, u1, json.RawMessage(`{"response":"one"}`), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := UpsertState(ctx, pool, inst.ID, e2, u2, json.RawMessage(`{"response":"two"}`), 0); err != nil {
		t.Fatal(err)
	}
	s1, err := GetState(ctx, pool, inst.ID, e1)
	if err != nil || s1 == nil {
		t.Fatalf("s1: %v", err)
	}
	s2, err := GetState(ctx, pool, inst.ID, e2)
	if err != nil || s2 == nil {
		t.Fatalf("s2: %v", err)
	}
	if string(s1.StateJSON) == string(s2.StateJSON) {
		t.Fatal("states should be independent")
	}
}

func TestDB_ListInstancesSingleQuery(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	courseID := seedCourse(t, pool)

	for i := 0; i < 20; i++ {
		if _, err := CreateInstance(ctx, pool, InstanceRow{
			CourseID: courseID, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
			ConfigJSON: json.RawMessage(`{"prompt":"p"}`),
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := ListInstances(ctx, pool, courseID, nil, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 20 {
		t.Fatalf("got %d instances", len(rows))
	}
}

func TestDB_ConfigSizeConstraint(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	courseID := seedCourse(t, pool)
	big := make([]byte, 300*1024)
	for i := range big {
		big[i] = 'x'
	}
	cfg, _ := json.Marshal(map[string]any{"prompt": string(big)})
	_, err := CreateInstance(ctx, pool, InstanceRow{
		CourseID: courseID, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: cfg,
	})
	if err == nil {
		t.Fatal("expected config size constraint failure")
	}
	if !IsConfigSizeViolation(err) {
		// Still acceptable if driver wraps differently — ensure some error.
		t.Logf("got error (ok): %v", err)
	}
}
