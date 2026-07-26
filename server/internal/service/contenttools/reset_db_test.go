package contenttools_test

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
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/user"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func resetTestPool(t *testing.T) *pgxpool.Pool {
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

func TestDB_ResetDryRunAndRestore(t *testing.T) {
	pool := resetTestPool(t)
	ctx := context.Background()

	courseID := uuid.New()
	courseCode := "C-" + strings.ToUpper(strings.ReplaceAll(courseID.String(), "-", "")[:6])
	if _, err := pool.Exec(ctx, `
INSERT INTO course.courses (id, course_code, title, content_tools_enabled)
VALUES ($1, $2, 'CT.4 reset', TRUE)
`, courseID, courseCode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.courses WHERE id = $1`, courseID)
	})

	em := "ct4-" + uuid.NewString()[:8] + "@example.test"
	ph, err := auth.HashPassword("Ct4-test-password-" + uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := uuid.Parse(u.ID)
	enrollmentID := uuid.New()
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (id, course_id, user_id, role, active)
VALUES ($1, $2, $3, 'student', TRUE)
`, enrollmentID, courseID, userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.course_enrollments WHERE id = $1`, enrollmentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM "user".users WHERE id = $1`, userID)
	})

	inst, err := ctrepo.CreateInstance(ctx, pool, ctrepo.InstanceRow{
		CourseID: courseID, HostKind: "syllabus", ToolID: "noop_probe", ToolVersion: "1.0.0",
		ConfigJSON: json.RawMessage(`{"prompt":"q"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ctrepo.UpsertState(ctx, pool, inst.ID, enrollmentID, userID, json.RawMessage(`{"response":"ans","attempts":1}`), 0); err != nil {
		t.Fatal(err)
	}

	var snapBefore int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.content_tool_state_resets WHERE course_id = $1`, courseID).Scan(&snapBefore)

	instID := inst.ID
	enrID := enrollmentID
	dry, err := ctsvc.ExecuteReset(ctx, pool, ctsvc.ResetRequest{
		CourseID: courseID, CourseCode: courseCode, ActorID: userID, ActorRole: "instructor",
		Scope: ctrepo.ResetScopeInstanceAll, InstanceID: &instID, DryRun: true, Notify: false, ToolID: "noop_probe",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !dry.DryRun || dry.AffectedCount != 1 {
		t.Fatalf("dry-run unexpected: %+v", dry)
	}
	var snapAfterDry int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.content_tool_state_resets WHERE course_id = $1`, courseID).Scan(&snapAfterDry)
	if snapAfterDry != snapBefore {
		t.Fatalf("dry-run mutated snapshots: before=%d after=%d", snapBefore, snapAfterDry)
	}

	real, err := ctsvc.ExecuteReset(ctx, pool, ctsvc.ResetRequest{
		CourseID: courseID, CourseCode: courseCode, ActorID: userID, ActorRole: "instructor",
		Scope: ctrepo.ResetScopeInstanceEnrollment, InstanceID: &instID, EnrollmentID: &enrID,
		DryRun: false, Notify: false, ToolID: "noop_probe", Reason: strPtr("test"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if real.AffectedCount != 1 || real.BatchID == nil {
		t.Fatalf("real reset unexpected: %+v", real)
	}
	st, err := ctrepo.GetState(ctx, pool, inst.ID, enrollmentID)
	if err != nil || st == nil {
		t.Fatalf("state after reset: %v", err)
	}
	if st.Status != ctsvc.StatusNotStarted || st.ResetCount != 1 {
		t.Fatalf("status=%s resetCount=%d", st.Status, st.ResetCount)
	}

	snaps, err := ctrepo.ListStateResets(ctx, pool, courseID, &instID, &enrID, 10)
	if err != nil || len(snaps) == 0 {
		t.Fatalf("snapshots: %v len=%d", err, len(snaps))
	}
	restored, snap, err := ctsvc.RestoreReset(ctx, pool, uuid.Nil, courseID, userID, snaps[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored == nil || snap == nil || snap.RestoredAt == nil {
		t.Fatal("expected restore")
	}
	if restored.Status != "in_progress" {
		t.Fatalf("restored status %s", restored.Status)
	}

	// Roster includes not_started peers: seed second enrollment without state.
	_, e2 := seedSecondEnrollment(t, pool, courseID)
	_ = e2
	rows, total, err := ctrepo.ListInstanceRoster(ctx, pool, ctrepo.RosterListParams{
		InstanceID: inst.ID, CourseID: courseID, Page: 1, PageSize: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total < 2 {
		t.Fatalf("expected >=2 roster rows, got %d", total)
	}
	var notStarted int
	for _, r := range rows {
		if r.Status == "not_started" {
			notStarted++
		}
	}
	if notStarted < 1 {
		t.Fatal("expected at least one not_started roster row")
	}
}

func seedSecondEnrollment(t *testing.T, pool *pgxpool.Pool, courseID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	em := "ct4b-" + uuid.NewString()[:8] + "@example.test"
	ph, err := auth.HashPassword("Ct4-test-password-" + uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, em, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := uuid.Parse(u.ID)
	enrollmentID := uuid.New()
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (id, course_id, user_id, role, active)
VALUES ($1, $2, $3, 'student', TRUE)
`, enrollmentID, courseID, userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM course.course_enrollments WHERE id = $1`, enrollmentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM "user".users WHERE id = $1`, userID)
	})
	return userID, enrollmentID
}

func strPtr(s string) *string { return &s }
