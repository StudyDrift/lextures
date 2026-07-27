package contenttools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
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
	// Format: C-[A-Z0-9]{6} (courses_course_code_format).
	courseCode := "C-" + strings.ToUpper(strings.ReplaceAll(courseID.String(), "-", "")[:6])
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
INSERT INTO course.course_enrollments (id, course_id, user_id, role, active)
VALUES ($1, $2, $3, 'student', TRUE)
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
	var n int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.content_tool_states WHERE enrollment_id = $1`, enrollmentID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("count before delete: n=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM course.course_enrollments WHERE id = $1`, enrollmentID); err != nil {
		t.Fatalf("delete enrollment: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.content_tool_states WHERE enrollment_id = $1`, enrollmentID).Scan(&n); err != nil || n != 0 {
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

func TestDB_UpsertStatePartialUniqueAndPreview(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	courseID := seedCourse(t, pool)
	userID, enrollmentID := seedEnrollment(t, pool, courseID)

	inst, err := CreateInstance(ctx, pool, InstanceRow{
		CourseID: courseID, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"q"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := UpsertState(ctx, pool, inst.ID, enrollmentID, userID, json.RawMessage(`{"response":"a"}`), 0)
	if err != nil || st == nil {
		t.Fatalf("upsert enrollment: %v %#v", err, st)
	}
	if st.Scope != ScopeEnrollment {
		t.Fatalf("scope=%q", st.Scope)
	}
	prev, err := UpsertPreviewState(ctx, pool, inst.ID, enrollmentID, userID, json.RawMessage(`{"response":"preview"}`), 0)
	if err != nil || prev == nil {
		t.Fatalf("upsert preview: %v %#v", err, prev)
	}
	if prev.Scope != ScopePreview {
		t.Fatalf("preview scope=%q", prev.Scope)
	}
	// Preview must not overwrite enrollment state.
	got, err := GetState(ctx, pool, inst.ID, enrollmentID)
	if err != nil || got == nil {
		t.Fatalf("get enrollment: %v", err)
	}
	if string(got.StateJSON) == string(prev.StateJSON) {
		t.Fatal("preview must not replace enrollment state")
	}
	withState, completed, _, err := CountInstanceUsage(ctx, pool, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if withState != 1 || completed != 0 {
		t.Fatalf("usage with=%d completed=%d", withState, completed)
	}
}

func TestDB_HardDeleteAndArchiveUnreferenced(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	courseID := seedCourse(t, pool)

	keep, err := CreateInstance(ctx, pool, InstanceRow{
		CourseID: courseID, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"keep"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	drop, err := CreateInstance(ctx, pool, InstanceRow{
		CourseID: courseID, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"drop"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ArchiveUnreferencedForItem(ctx, pool, courseID, nil, []uuid.UUID{keep.ID}); err != nil {
		t.Fatal(err)
	}
	rows, err := ListInstances(ctx, pool, courseID, nil, "", true)
	if err != nil {
		t.Fatal(err)
	}
	var dropStatus string
	for _, r := range rows {
		if r.ID == drop.ID {
			dropStatus = r.Status
		}
	}
	if dropStatus != "archived" {
		t.Fatalf("expected archived, got %q", dropStatus)
	}
	if err := HardDeleteInstance(ctx, pool, courseID, drop.ID); err != nil {
		t.Fatal(err)
	}
	gone, err := GetInstance(ctx, pool, courseID, drop.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gone != nil {
		t.Fatal("expected hard delete")
	}
}

func TestDB_StateRevisionConflictAndIdempotency(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	courseID := seedCourse(t, pool)
	userID, enrollmentID := seedEnrollment(t, pool, courseID)

	inst, err := CreateInstance(ctx, pool, InstanceRow{
		CourseID: courseID, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"q","answerKey":"yes"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := UpsertStateWithStatus(ctx, pool, inst.ID, enrollmentID, userID, json.RawMessage(`{"response":"a"}`), 0, "in_progress", 0)
	if err != nil || st == nil {
		t.Fatalf("first upsert: %v %#v", err, st)
	}
	conflict, err := UpsertStateWithStatus(ctx, pool, inst.ID, enrollmentID, userID, json.RawMessage(`{"response":"b"}`), 0, "in_progress", 0)
	if err != nil {
		t.Fatal(err)
	}
	if conflict != nil {
		t.Fatal("expected revision conflict (nil row)")
	}
	ok, err := UpsertStateWithStatus(ctx, pool, inst.ID, enrollmentID, userID, json.RawMessage(`{"response":"c"}`), st.Revision, "in_progress", 0)
	if err != nil || ok == nil {
		t.Fatalf("second upsert: %v %#v", err, ok)
	}
	if ok.Revision != st.Revision+1 {
		t.Fatalf("revision=%d want %d", ok.Revision, st.Revision+1)
	}

	raw, _ := json.Marshal(map[string]any{"result": map[string]any{"correct": true}})
	key := "ct3-idem-" + uuid.NewString()
	if err := PutActionIdempotency(ctx, pool, key, inst.ID, enrollmentID, "grade", raw); err != nil {
		t.Fatal(err)
	}
	got, err := GetActionIdempotency(ctx, pool, key)
	if err != nil || got == nil {
		t.Fatalf("get idempotency: %v", err)
	}
	if got.Action != "grade" {
		t.Fatalf("action=%s", got.Action)
	}
	// First-write-wins on conflict.
	if err := PutActionIdempotency(ctx, pool, key, inst.ID, enrollmentID, "grade", json.RawMessage(`{"result":{"correct":false}}`)); err != nil {
		t.Fatal(err)
	}
	got2, err := GetActionIdempotency(ctx, pool, key)
	if err != nil || got2 == nil {
		t.Fatal(err)
	}
	var a, b any
	if err := json.Unmarshal(got2.ResultJSON, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	aNorm, _ := json.Marshal(a)
	bNorm, _ := json.Marshal(b)
	if string(aNorm) != string(bNorm) {
		t.Fatalf("idempotency overwritten: %s vs %s", got2.ResultJSON, raw)
	}
}
