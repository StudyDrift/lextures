package quizgradebook

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
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/repos/user"
)

type fixture struct {
	pool      *pgxpool.Pool
	courseID  uuid.UUID
	quizID    uuid.UUID
	studentID uuid.UUID
}

func seed(t *testing.T, ctx context.Context, pool *pgxpool.Pool, policy string) fixture {
	t.Helper()
	ph, err := auth.HashPassword("password1230password1230")
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, "qgb-"+uuid.NewString()+"@example.com", ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	studentID, _ := uuid.Parse(u.ID)
	cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:6])
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Quiz gradebook sync', $2) RETURNING id
`, cc, studentID).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	var moduleID, quizID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 0, 'module', 'Mod', NULL, TRUE) RETURNING id`, courseID).Scan(&moduleID); err != nil {
		t.Fatalf("module: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 1, 'quiz', 'Quiz', $2, TRUE) RETURNING id`, courseID, moduleID).Scan(&quizID); err != nil {
		t.Fatalf("quiz item: %v", err)
	}
	qJSON, _ := json.Marshal([]map[string]any{
		{"id": uuid.NewString(), "prompt": "2+2?", "questionType": "multiple_choice", "choices": []string{"4", "5"}, "correctChoiceIndex": 0, "points": 1},
		{"id": uuid.NewString(), "prompt": "Why?", "questionType": "essay", "points": 1},
	})
	if _, err := pool.Exec(ctx, `
INSERT INTO course.module_quizzes (structure_item_id, markdown, questions_json, grade_attempt_policy)
VALUES ($1, '', $2::jsonb, $3)`, quizID, qJSON, policy); err != nil {
		t.Fatalf("module quiz: %v", err)
	}
	return fixture{pool: pool, courseID: courseID, quizID: quizID, studentID: studentID}
}

func submitAttempt(t *testing.T, ctx context.Context, f fixture, number int32, earned float64, needsManual bool) {
	t.Helper()
	att, err := quizattempts.InsertAttempt(ctx, f.pool, quizattempts.InsertAttemptParams{
		CourseID: f.courseID, StructureItemID: f.quizID, StudentUserID: f.studentID, AttemptNumber: number,
	})
	if err != nil {
		t.Fatalf("attempt: %v", err)
	}
	resp := json.RawMessage(`{}`)
	rows := []quizattempts.ResponseRow{{
		QuestionIndex: 0, QuestionID: "q0", QuestionType: "multiple_choice", ResponseJSON: resp,
		PointsAwarded: earned, MaxPoints: 1,
	}}
	if needsManual {
		// Essay answers are not auto-graded: is_correct stays NULL until staff grade them.
		rows = append(rows, quizattempts.ResponseRow{
			QuestionIndex: 1, QuestionID: "q1", QuestionType: "essay", ResponseJSON: json.RawMessage(`{"textAnswer":"x"}`), MaxPoints: 1,
		})
	}
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := quizattempts.ReplaceResponses(ctx, tx, att.ID, rows); err != nil {
		t.Fatalf("responses: %v", err)
	}
	if ok, err := quizattempts.FinalizeAttemptSubmitted(ctx, tx, quizattempts.FinalizeSubmitParams{
		AttemptID: att.ID, SubmittedAt: time.Now().UTC().Add(time.Duration(number) * time.Second),
		PointsEarned: earned, PointsPossible: float64(len(rows)), ScorePercent: 50,
	}); err != nil || !ok {
		t.Fatalf("finalize ok=%v err=%v", ok, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func cell(t *testing.T, ctx context.Context, f fixture) (float64, bool) {
	t.Helper()
	var pts float64
	err := f.pool.QueryRow(ctx, `
SELECT points_earned::float8 FROM course.course_grades WHERE student_user_id = $1 AND module_item_id = $2`,
		f.studentID, f.quizID).Scan(&pts)
	if err != nil {
		return 0, false
	}
	return pts, true
}

func testPool(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, ctx
}

func TestSyncStudentCell_WritesAutoGradedScore_Pg(t *testing.T) {
	pool, ctx := testPool(t)
	f := seed(t, ctx, pool, "highest")
	submitAttempt(t, ctx, f, 1, 1, false)

	synced, err := SyncStudentCell(ctx, pool, f.courseID, f.studentID, f.quizID)
	if err != nil || !synced {
		t.Fatalf("synced=%v err=%v", synced, err)
	}
	if pts, ok := cell(t, ctx, f); !ok || pts != 1 {
		t.Fatalf("cell pts=%v ok=%v want 1", pts, ok)
	}
}

func TestSyncStudentCell_SkipsAttemptsNeedingManualGrading_Pg(t *testing.T) {
	pool, ctx := testPool(t)
	f := seed(t, ctx, pool, "highest")
	submitAttempt(t, ctx, f, 1, 1, true)

	synced, err := SyncStudentCell(ctx, pool, f.courseID, f.studentID, f.quizID)
	if err != nil || synced {
		t.Fatalf("synced=%v err=%v want not synced", synced, err)
	}
	if _, ok := cell(t, ctx, f); ok {
		t.Fatal("partial score must not reach the gradebook while manual grading is pending")
	}
}

func TestSyncStudentCell_HonoursAttemptPolicy_Pg(t *testing.T) {
	pool, ctx := testPool(t)
	f := seed(t, ctx, pool, "highest")
	submitAttempt(t, ctx, f, 1, 1, false)
	submitAttempt(t, ctx, f, 2, 0, false)

	if _, err := SyncStudentCell(ctx, pool, f.courseID, f.studentID, f.quizID); err != nil {
		t.Fatal(err)
	}
	if pts, ok := cell(t, ctx, f); !ok || pts != 1 {
		t.Fatalf("highest policy: pts=%v ok=%v want 1", pts, ok)
	}
}
