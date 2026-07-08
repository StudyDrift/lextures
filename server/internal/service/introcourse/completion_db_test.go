package introcourse

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
)

func TestProgress_NotEnrolled_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	studentID := insertTestUser(t, pool, ctx)
	prog, err := LoadProgress(ctx, pool, cfg, course.ID, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if prog.Enrolled {
		t.Fatal("expected not enrolled without enrollment row")
	}
}

func TestProgress_PartialModules_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	studentID := insertTestUser(t, pool, ctx)
	enrollStudent(t, pool, ctx, course.ID, studentID)

	quizSlugs := []string{
		"m1.welcome.knowledge-check",
		"m2.core.knowledge-check",
		"m3.patterns.knowledge-check",
	}
	for _, slug := range quizSlugs {
		submitQuizAttempt(t, pool, ctx, course.ID, studentID, slug)
	}

	prog, err := LoadProgress(ctx, pool, cfg, course.ID, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if !prog.Enrolled {
		t.Fatal("expected enrolled")
	}
	if prog.ModulesComplete != 3 {
		t.Fatalf("modulesComplete: got %d want 3", prog.ModulesComplete)
	}
	if prog.ModulesTotal != 7 {
		t.Fatalf("modulesTotal: got %d want 7", prog.ModulesTotal)
	}
	if prog.Percent != 42 {
		t.Fatalf("percent: got %d want 42", prog.Percent)
	}
	if prog.NextItem == nil {
		t.Fatal("expected nextItem")
	}
}

func TestRecheckCompletion_Idempotent_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{
		IntroCourseEnabled:      true,
		LearnerProfileEnabled:   true,
		FFCompletionCredentials: true,
		PublicWebOrigin:         "http://localhost:5173",
		JWTSecret:               "01234567890123456789012345678901",
	}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	studentID := insertTestUser(t, pool, ctx)
	enrollStudent(t, pool, ctx, course.ID, studentID)
	qualifyStudentForCompletion(t, pool, ctx, course.ID, studentID, cfg)

	first, err := RecheckCompletion(ctx, pool, cfg, course.ID, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if first.CompletedAt == nil {
		t.Fatal("expected completion after qualifying grades (may have completed on last grade write)")
	}

	second, err := RecheckCompletion(ctx, pool, cfg, course.ID, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if second.JustCompleted {
		t.Fatal("expected idempotent recheck")
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM settings.intro_course_completions WHERE user_id = $1`, studentID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("completion rows: got %d want 1", count)
	}

	var credCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM credentials.issued_credentials
WHERE recipient_id = $1 AND source_type = 'course' AND source_id = $2
`, studentID, course.ID).Scan(&credCount); err != nil {
		t.Fatal(err)
	}
	if credCount != 1 {
		t.Fatalf("credential rows: got %d want 1", credCount)
	}
}

func TestRecheckCompletion_NoCredentialWhenFlagOff_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{
		IntroCourseEnabled:      true,
		LearnerProfileEnabled:   true,
		FFCompletionCredentials: false,
	}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	studentID := insertTestUser(t, pool, ctx)
	enrollStudent(t, pool, ctx, course.ID, studentID)
	qualifyStudentForCompletion(t, pool, ctx, course.ID, studentID, cfg)

	prog, err := RecheckCompletion(ctx, pool, cfg, course.ID, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if prog.CompletedAt == nil {
		t.Fatal("expected completion recorded")
	}

	var credCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM credentials.issued_credentials WHERE recipient_id = $1
`, studentID).Scan(&credCount); err != nil {
		t.Fatal(err)
	}
	if credCount != 0 {
		t.Fatalf("credential rows: got %d want 0", credCount)
	}
}

func TestShouldNudgeIntroCourse_SuppressesCompleters_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	studentID := insertTestUser(t, pool, ctx)
	enrollStudent(t, pool, ctx, course.ID, studentID)

	nudge, err := ShouldNudgeIntroCourse(ctx, pool, cfg, studentID)
	if err != nil || !nudge {
		t.Fatalf("incomplete student should be nudged: nudge=%v err=%v", nudge, err)
	}

	qualifyStudentForCompletion(t, pool, ctx, course.ID, studentID, cfg)
	if _, err := RecheckCompletion(ctx, pool, cfg, course.ID, studentID); err != nil {
		t.Fatal(err)
	}

	nudge, err = ShouldNudgeIntroCourse(ctx, pool, cfg, studentID)
	if err != nil || nudge {
		t.Fatalf("completer should not be nudged: nudge=%v err=%v", nudge, err)
	}
}

func TestLoadAnalytics_Pg(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	svc := New(pool)
	cfg := config.Config{IntroCourseEnabled: true, LearnerProfileEnabled: true}

	course, err := svc.EnsureProvisioned(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	studentID := insertTestUser(t, pool, ctx)
	enrollStudent(t, pool, ctx, course.ID, studentID)
	submitQuizAttempt(t, pool, ctx, course.ID, studentID, "m1.welcome.knowledge-check")

	analytics, err := LoadAnalytics(ctx, pool, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if analytics.Enrolled < 1 {
		t.Fatalf("enrolled: %d", analytics.Enrolled)
	}
	if len(analytics.PerModuleFunnel) != 7 {
		t.Fatalf("funnel modules: got %d want 7", len(analytics.PerModuleFunnel))
	}
}

func enrollStudent(t *testing.T, pool *pgxpool.Pool, ctx context.Context, courseID, userID uuid.UUID) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active, state)
VALUES ($1, $2, 'student', TRUE, 'active')
ON CONFLICT (course_id, user_id, role) DO UPDATE SET active = TRUE, state = 'active'
`, courseID, userID); err != nil {
		t.Fatal(err)
	}
}

func submitQuizAttempt(t *testing.T, pool *pgxpool.Pool, ctx context.Context, courseID, studentID uuid.UUID, slug string) {
	t.Helper()
	var itemID uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT structure_item_id FROM settings.intro_course_items WHERE slug = $1
`, slug).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.quiz_attempts (
    course_id, structure_item_id, student_user_id, attempt_number, status,
    points_earned, points_possible, score_percent, submitted_at
) VALUES ($1, $2, $3, 1, 'submitted', 3, 3, 100, NOW())
`, courseID, itemID, studentID); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{IntroCourseEnabled: true}
	if err := OnQuizAttempt(ctx, pool, cfg, courseID, studentID, itemID); err != nil {
		t.Fatal(err)
	}
}

func qualifyStudentForCompletion(t *testing.T, pool *pgxpool.Pool, ctx context.Context, courseID, studentID uuid.UUID, cfg config.Config) {
	t.Helper()
	quizSlugs := []string{
		"m1.welcome.knowledge-check",
		"m2.core.knowledge-check",
		"m3.patterns.knowledge-check",
		"m4.profile.knowledge-check",
		"m5.mobile.knowledge-check",
		"m6.canvas.knowledge-check",
		"m7.finish.knowledge-check",
	}
	for _, slug := range quizSlugs {
		submitQuizAttempt(t, pool, ctx, courseID, studentID, slug)
	}

	var capstoneID uuid.UUID
	if err := pool.QueryRow(ctx, `
SELECT structure_item_id FROM settings.intro_course_items WHERE slug = $1
`, CapstoneSlug).Scan(&capstoneID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.module_assignment_submissions (course_id, module_item_id, submitted_by, body_text)
VALUES ($1, $2, $3, 'Reflecting on my Lextures journey.')
`, courseID, capstoneID, studentID); err != nil {
		t.Fatal(err)
	}
	if _, err := OnAssignmentSubmit(ctx, pool, cfg, courseID, studentID, capstoneID, icrepo.CourseCode); err != nil {
		t.Fatal(err)
	}
}