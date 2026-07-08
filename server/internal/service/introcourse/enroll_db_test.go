package introcourse

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/config"
	pauth "github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestEnsureEnrollment_SignupEnrollsStudent_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true}
	if _, err := svc.EnsureProvisioned(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	courseID, ok, err := svc.CourseID(ctx)
	if err != nil || !ok {
		t.Fatalf("course: ok=%v err=%v", ok, err)
	}

	ph, _ := pauth.HashPassword("test-password-long-enough-12345")
	email := "ic02-signup-" + uuid.NewString() + "@test.example"
	row, err := user.InsertUser(ctx, pool, email, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	uid, err := uuid.Parse(row.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.EnsureEnrollment(ctx, cfg, pool, uid, PathSignup); err != nil {
		t.Fatalf("first enroll: %v", err)
	}
	if err := svc.EnsureEnrollment(ctx, cfg, pool, uid, PathSignup); err != nil {
		t.Fatalf("second enroll: %v", err)
	}

	var role string
	err = pool.QueryRow(ctx, `
SELECT role FROM course.course_enrollments WHERE course_id = $1 AND user_id = $2
`, courseID, uid).Scan(&role)
	if err != nil {
		t.Fatal(err)
	}
	if role != "student" {
		t.Fatalf("expected student role, got %q", role)
	}
	var dupCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_enrollments WHERE course_id = $1 AND user_id = $2 AND role = 'student'
`, courseID, uid).Scan(&dupCount); err != nil {
		t.Fatal(err)
	}
	if dupCount != 1 {
		t.Fatalf("expected exactly one student enrollment, got %d", dupCount)
	}
}

func TestEnsureEnrollment_SkipsParentAndSystem_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true}
	if _, err := svc.EnsureProvisioned(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	courseID, ok, _ := svc.CourseID(ctx)
	if !ok {
		t.Fatal("no course")
	}

	ph, _ := pauth.HashPassword("test-password-long-enough-12345")
	parentEmail := "ic02-parent-" + uuid.NewString() + "@test.example"
	parent, err := user.InsertUser(ctx, pool, parentEmail, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	parentID, _ := uuid.Parse(parent.ID)
	if _, err := pool.Exec(ctx, `UPDATE "user".users SET account_type = 'parent' WHERE id = $1`, parentID); err != nil {
		t.Fatal(err)
	}
	if err := svc.EnsureEnrollment(ctx, cfg, pool, parentID, PathSignup); err != nil {
		t.Fatal(err)
	}
	var parentEnrolled int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_enrollments WHERE course_id = $1 AND user_id = $2
`, courseID, parentID).Scan(&parentEnrolled); err != nil {
		t.Fatal(err)
	}
	if parentEnrolled != 0 {
		t.Fatal("parent should not be enrolled")
	}

	if err := svc.EnsureEnrollment(ctx, cfg, pool, SystemUserID, PathSignup); err != nil {
		t.Fatal(err)
	}
	var systemStudent int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_enrollments
WHERE course_id = $1 AND user_id = $2 AND role = 'student'
`, courseID, SystemUserID).Scan(&systemStudent); err != nil {
		t.Fatal(err)
	}
	if systemStudent != 0 {
		t.Fatal("system instructor must not be enrolled as student")
	}
}

func TestEnsureEnrollment_FlagOffNoOp_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true}
	if _, err := svc.EnsureProvisioned(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	courseID, ok, _ := svc.CourseID(ctx)
	if !ok {
		t.Fatal("no course")
	}

	ph, _ := pauth.HashPassword("test-password-long-enough-12345")
	email := "ic02-flagoff-" + uuid.NewString() + "@test.example"
	row, err := user.InsertUser(ctx, pool, email, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	uid, _ := uuid.Parse(row.ID)

	if err := svc.EnsureEnrollment(ctx, config.Config{IntroCourseEnabled: false}, pool, uid, PathSignup); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_enrollments WHERE course_id = $1 AND user_id = $2
`, courseID, uid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("flag off must not enroll")
	}
}

func TestRunBackfill_EnrollsExistingUsers_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true}
	if _, err := svc.EnsureProvisioned(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	courseID, ok, _ := svc.CourseID(ctx)
	if !ok {
		t.Fatal("no course")
	}

	if _, err := pool.Exec(ctx, `
UPDATE settings.intro_course_backfill
SET completed_at = NULL, last_user_id = NULL, enrolled_count = 0, started_at = NULL
WHERE id = TRUE`); err != nil {
		t.Fatal(err)
	}

	ph, _ := pauth.HashPassword("test-password-long-enough-12345")
	var userIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		email := "ic02-backfill-" + uuid.NewString() + "@test.example"
		row, err := user.InsertUser(ctx, pool, email, ph, nil)
		if err != nil {
			t.Fatal(err)
		}
		uid, _ := uuid.Parse(row.ID)
		userIDs = append(userIDs, uid)
	}

	if err := svc.RunBackfill(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	st, err := svc.BackfillStatus(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if st.CompletedAt == nil {
		t.Fatal("expected completed backfill")
	}

	for _, uid := range userIDs {
		var role string
		err := pool.QueryRow(ctx, `
SELECT role FROM course.course_enrollments WHERE course_id = $1 AND user_id = $2
`, courseID, uid).Scan(&role)
		if err != nil {
			t.Fatalf("user %s not enrolled: %v", uid, err)
		}
		if role != "student" {
			t.Fatalf("user %s role=%q", uid, role)
		}
	}

	if err := svc.RunBackfill(ctx, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureEnrollment_InTxWithSignup_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true}
	if _, err := svc.EnsureProvisioned(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	courseID, ok, _ := svc.CourseID(ctx)
	if !ok {
		t.Fatal("no course")
	}

	ph, _ := pauth.HashPassword("test-password-long-enough-12345")
	email := "ic02-tx-" + uuid.NewString() + "@test.example"

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	row, err := user.InsertUserTx(ctx, tx, email, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	uid, _ := uuid.Parse(row.ID)
	if err := svc.EnsureEnrollment(ctx, cfg, tx, uid, PathSignup); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	var role string
	err = pool.QueryRow(ctx, `
SELECT role FROM course.course_enrollments WHERE course_id = $1 AND user_id = $2
`, courseID, uid).Scan(&role)
	if err == pgx.ErrNoRows {
		t.Fatal("expected enrollment committed with user")
	}
	if err != nil {
		t.Fatal(err)
	}
	if role != "student" {
		t.Fatalf("got role %q", role)
	}
}