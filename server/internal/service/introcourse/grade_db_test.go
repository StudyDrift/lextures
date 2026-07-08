package introcourse

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	pauth "github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/user"
)

func TestSyncContent_GradingConfig_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	var assignmentCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.course_structure_items
WHERE course_id = $1 AND kind = 'assignment' AND NOT archived
`, course.ID).Scan(&assignmentCount); err != nil {
		t.Fatal(err)
	}
	if assignmentCount < 2 {
		t.Fatalf("expected at least 2 assignments, got %d", assignmentCount)
	}

	var quizPoints int
	if err := pool.QueryRow(ctx, `
SELECT m.points_worth FROM course.module_quizzes m
INNER JOIN settings.intro_course_items ici ON ici.structure_item_id = m.structure_item_id
WHERE ici.slug = 'm1.welcome.knowledge-check'
`).Scan(&quizPoints); err != nil {
		t.Fatal(err)
	}
	if quizPoints != 3 {
		t.Fatalf("expected quiz points_worth=3, got %d", quizPoints)
	}

	var quizPolicy string
	if err := pool.QueryRow(ctx, `
SELECT grade_policy FROM settings.intro_course_items WHERE slug = 'm7.finish.capstone'
`).Scan(&quizPolicy); err != nil {
		t.Fatal(err)
	}
	if quizPolicy != GradePolicyGraderAgent {
		t.Fatalf("capstone policy: got %q", quizPolicy)
	}

	var quizWeight float64
	if err := pool.QueryRow(ctx, `
SELECT weight_percent FROM course.assignment_groups
WHERE course_id = $1 AND name = 'Quizzes'
`, course.ID).Scan(&quizWeight); err != nil {
		t.Fatal(err)
	}
	if quizWeight != 50 {
		t.Fatalf("expected Quizzes weight 50, got %v", quizWeight)
	}
}

func TestOnQuizAttempt_WritesGrade_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	var itemID uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT structure_item_id FROM settings.intro_course_items WHERE slug = 'm1.welcome.knowledge-check'
`).Scan(&itemID); err != nil {
		t.Fatal(err)
	}

	studentID := insertTestUser(t, pool, ctx)

	var attemptID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.quiz_attempts (
    course_id, structure_item_id, student_user_id, attempt_number, status,
    points_earned, points_possible, score_percent, submitted_at
) VALUES ($1, $2, $3, 1, 'submitted', 2, 3, 66.67, NOW())
RETURNING id
`, course.ID, itemID, studentID).Scan(&attemptID); err != nil {
		t.Fatal(err)
	}
	_ = attemptID

	if err := OnQuizAttempt(ctx, pool, cfg, course.ID, studentID, itemID); err != nil {
		t.Fatal(err)
	}

	var points float64
	if err := pool.QueryRow(ctx, `
SELECT points_earned FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, course.ID, studentID, itemID).Scan(&points); err != nil {
		t.Fatal(err)
	}
	if points != 2 {
		t.Fatalf("expected grade 2, got %v", points)
	}

	// Second higher attempt — keep-highest should update grade.
	if _, err := pool.Exec(ctx, `
INSERT INTO course.quiz_attempts (
    course_id, structure_item_id, student_user_id, attempt_number, status,
    points_earned, points_possible, score_percent, submitted_at
) VALUES ($1, $2, $3, 2, 'submitted', 3, 3, 100, NOW())
`, course.ID, itemID, studentID); err != nil {
		t.Fatal(err)
	}
	if err := OnQuizAttempt(ctx, pool, cfg, course.ID, studentID, itemID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
SELECT points_earned FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, course.ID, studentID, itemID).Scan(&points); err != nil {
		t.Fatal(err)
	}
	if points != 3 {
		t.Fatalf("expected keep-highest grade 3, got %v", points)
	}
}

func TestOnAssignmentSubmit_FullCredit_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	var itemID uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT structure_item_id FROM settings.intro_course_items WHERE slug = 'm2.try-it.dashboard'
`).Scan(&itemID); err != nil {
		t.Fatal(err)
	}

	studentID := insertTestUser(t, pool, ctx)
	if _, err := pool.Exec(ctx, `
INSERT INTO course.module_assignment_submissions (course_id, module_item_id, submitted_by, body_text)
VALUES ($1, $2, $3, 'I opened my dashboard.')
`, course.ID, itemID, studentID); err != nil {
		t.Fatal(err)
	}

	if _, err := OnAssignmentSubmit(ctx, pool, cfg, course.ID, studentID, itemID, CourseCode); err != nil {
		t.Fatal(err)
	}

	var points float64
	if err := pool.QueryRow(ctx, `
SELECT points_earned FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, course.ID, studentID, itemID).Scan(&points); err != nil {
		t.Fatal(err)
	}
	if points != 5 {
		t.Fatalf("expected full credit 5, got %v", points)
	}
}

func TestSyncContent_PreservesExistingGrades_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	var itemID uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT structure_item_id FROM settings.intro_course_items WHERE slug = 'm2.try-it.dashboard'
`).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	studentID := insertTestUser(t, pool, ctx)
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_grades (course_id, student_user_id, module_item_id, points_earned, posted_at)
VALUES ($1, $2, $3, 5, NOW())
`, course.ID, studentID, itemID); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.SyncContentForCourse(ctx, cfg, course.ID); err != nil {
		t.Fatal(err)
	}

	var points float64
	if err := pool.QueryRow(ctx, `
SELECT points_earned FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, course.ID, studentID, itemID).Scan(&points); err != nil {
		t.Fatal(err)
	}
	if points != 5 {
		t.Fatalf("re-sync altered grade: got %v", points)
	}
}

func insertTestUser(t *testing.T, pool *pgxpool.Pool, ctx context.Context) uuid.UUID {
	t.Helper()
	ph, err := pauth.HashPassword("test-password-long-enough-12345")
	if err != nil {
		t.Fatal(err)
	}
	email := "ic04-" + uuid.NewString() + "@test.example"
	row, err := user.InsertUser(ctx, pool, email, ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	uid, err := uuid.Parse(row.ID)
	if err != nil {
		t.Fatal(err)
	}
	return uid
}