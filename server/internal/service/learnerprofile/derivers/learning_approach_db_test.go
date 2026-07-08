package derivers

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
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	learnerprofileservice "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

func TestLearningApproachDeriver_InsufficientData_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openLearningApproachPool(t, ctx)
	defer pool.Close()

	userID := insertLearningApproachUser(t, ctx, pool)
	defer deleteLearningApproachUser(ctx, pool, userID)

	fix := seedLearningApproachCourse(t, ctx, pool, userID)
	insertSubmittedQuizAttempt(t, ctx, pool, fix, userID, 1, 1, 60, time.Now().UTC())
	insertSubmittedQuizAttempt(t, ctx, pool, fix, userID, 1, 2, 70, time.Now().UTC())

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	deriver := LearningApproachDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}
	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "insufficient_data" {
		t.Fatalf("state=%q want insufficient_data", result.State)
	}
}

func TestLearningApproachDeriver_FullFacet_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openLearningApproachPool(t, ctx)
	defer pool.Close()

	userID := insertLearningApproachUser(t, ctx, pool)
	defer deleteLearningApproachUser(ctx, pool, userID)

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	fix := seedLearningApproachCourse(t, ctx, pool, userID)

	started := fixedNow.AddDate(0, 0, -7)
	attempt1 := insertSubmittedQuizAttempt(t, ctx, pool, fix, userID, 1, 1, 52, started)
	attempt2 := insertSubmittedQuizAttempt(t, ctx, pool, fix, userID, 1, 2, 70, started.Add(2*time.Hour))
	attempt3 := insertSubmittedQuizAttempt(t, ctx, pool, fix, userID, 2, 1, 48, started.Add(4*time.Hour))
	attempt4 := insertSubmittedQuizAttempt(t, ctx, pool, fix, userID, 2, 2, 66, started.Add(6*time.Hour))
	insertEarlyHint(t, ctx, pool, attempt1, "q1", started.Add(4*time.Second))
	insertEarlyHint(t, ctx, pool, attempt2, "q1", started.Add(2*time.Hour+6*time.Second))
	insertEarlyHint(t, ctx, pool, attempt3, "q2", started.Add(4*time.Hour+3*time.Second))
	insertEarlyHint(t, ctx, pool, attempt4, "q2", started.Add(6*time.Hour+5*time.Second))
	insertSubmittedQuizAttempt(t, ctx, pool, fix, userID, 3, 1, 75, started.AddDate(0, 0, -1))

	for i := 0; i < 12; i++ {
		pageID := "p" + string(rune('a'+i))
		insertLearningApproachNotebookPage(t, ctx, pool, userID, globalStudentNotebookKey, "Study note "+pageID, pageID)
	}
	for i := 0; i < 10; i++ {
		insertLearningApproachNotebookTask(t, ctx, pool, userID, globalStudentNotebookKey, "task-"+string(rune('a'+i)))
	}

	deriver := LearningApproachDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}
	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "ok" {
		t.Fatalf("state=%q", result.State)
	}

	var summary LearningApproachSummary
	if err := json.Unmarshal(result.Summary, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Persistence.Level != "high" || !summary.Persistence.Productive {
		t.Fatalf("persistence=%+v", summary.Persistence)
	}
	if summary.HelpSeeking.Style != "early-reliance" {
		t.Fatalf("helpSeeking=%+v", summary.HelpSeeking)
	}
	if summary.Consolidation.Level != "active" || summary.Consolidation.NotebookActions < 20 {
		t.Fatalf("consolidation=%+v", summary.Consolidation)
	}

	svc := learnerprofileservice.New(pool, deriver)
	if err := svc.RecomputeIncremental(ctx, userID, "learning_approach"); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.GetFacet(ctx, userID, "learning_approach")
	if err != nil || detail == nil {
		t.Fatalf("get facet: %v %+v", err, detail)
	}
	if detail.Facet.State != "ok" || len(detail.Insights) == 0 {
		t.Fatalf("stored facet=%+v insights=%d", detail.Facet, len(detail.Insights))
	}
}

type learningApproachCourseFixture struct {
	courseID uuid.UUID
	quizIDs  map[int]uuid.UUID
}

func openLearningApproachPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	return pool
}

func insertLearningApproachUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	email := "learning-approach-" + userID.String() + "@e.invalid"
	if _, err := pool.Exec(ctx, `
INSERT INTO "user".users (id, email, password_hash, display_name)
VALUES ($1, $2, 'hash', 'Learning Approach Tester')
`, userID, email); err != nil {
		t.Fatal(err)
	}
	return userID
}

func deleteLearningApproachUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) {
	_, _ = pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, userID)
}

func seedLearningApproachCourse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) learningApproachCourseFixture {
	t.Helper()
	cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6])
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id)
VALUES ($1, 'Learning approach test', $2) RETURNING id
`, cc, userID).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active)
VALUES ($1, $2, 'student', TRUE)
`, courseID, userID); err != nil {
		t.Fatal(err)
	}
	var moduleID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 0, 'module', 'Mod', NULL, TRUE) RETURNING id
`, courseID).Scan(&moduleID); err != nil {
		t.Fatal(err)
	}
	quizIDs := make(map[int]uuid.UUID)
	for i := 1; i <= 3; i++ {
		var quizID uuid.UUID
		if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, $2, 'quiz', $3, $4, TRUE) RETURNING id
`, courseID, i, "Quiz "+string(rune('A'+i-1)), moduleID).Scan(&quizID); err != nil {
			t.Fatal(err)
		}
		quizIDs[i] = quizID
	}
	return learningApproachCourseFixture{courseID: courseID, quizIDs: quizIDs}
}

func insertSubmittedQuizAttempt(
	t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	fix learningApproachCourseFixture, userID uuid.UUID,
	quizIndex, attemptNumber int, score float32, started time.Time,
) uuid.UUID {
	t.Helper()
	quizID, ok := fix.quizIDs[quizIndex]
	if !ok {
		t.Fatalf("unknown quiz index %d", quizIndex)
	}
	var attemptID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.quiz_attempts (
    course_id, structure_item_id, student_user_id, attempt_number, status,
    started_at, submitted_at, score_percent
)
VALUES ($1, $2, $3, $4, 'submitted', $5, $5, $6)
RETURNING id
`, fix.courseID, quizID, userID, attemptNumber, started, score).Scan(&attemptID); err != nil {
		t.Fatal(err)
	}
	return attemptID
}

func insertEarlyHint(t *testing.T, ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID, questionID string, requestedAt time.Time) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO course.hint_requests (attempt_id, question_id, hint_level, requested_at)
VALUES ($1, $2, 1, $3)
`, attemptID, questionID, requestedAt); err != nil {
		t.Fatal(err)
	}
}

func insertLearningApproachNotebookPage(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseCode, content, pageID string) {
	t.Helper()
	upsertNotebookPages(t, ctx, pool, userID, courseCode, []notebookPage{
		{ID: pageID, Kind: "page", ContentMd: content},
	})
}

func insertLearningApproachNotebookTask(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseCode, pageID string) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
INSERT INTO analytics.student_notebook_tasks (id, user_id, course_code, notebook_page_id, task_text)
VALUES ($1, $2, $3, $4, $5)
`, uuid.New(), userID, courseCode, pageID, "Review notes"); err != nil {
		t.Fatal(err)
	}
}