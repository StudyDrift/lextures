package coursefinalgrade

import (
	"context"
	"encoding/json"
	"math"
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
	"github.com/lextures/lextures/server/internal/repos/conditionalrelease"
	"github.com/lextures/lextures/server/internal/repos/studentprogress"
	"github.com/lextures/lextures/server/internal/repos/user"
)

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

// Issue #736: assignment 8/10 plus a quiz with two one-point questions and no "points worth" (1/2)
// must read 75% (9/12) — the gradebook course grade — not 80% (quiz dropped) or 65% (unweighted mean).
func TestPercentForStudent_QuizWithoutPointsWorth_Pg(t *testing.T) {
	pool, ctx := testPool(t)
	ph, err := auth.HashPassword("password1230password1230")
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, "cfg-"+uuid.NewString()+"@example.com", ph, nil)
	if err != nil {
		t.Fatal(err)
	}
	studentID, _ := uuid.Parse(u.ID)
	cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:6])
	var courseID, moduleID, assignID, quizID uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Final grade', $2) RETURNING id`, cc, studentID).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	insertItem := func(order int, kind, title string, parent *uuid.UUID, dst *uuid.UUID) {
		t.Helper()
		if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, $2, $3, $4, $5, TRUE) RETURNING id`, courseID, order, kind, title, parent).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	insertItem(0, "module", "Mod", nil, &moduleID)
	insertItem(1, "assignment", "QA Assignment", &moduleID, &assignID)
	insertItem(2, "quiz", "QA Quiz", &moduleID, &quizID)
	if _, err := pool.Exec(ctx, `INSERT INTO course.module_assignments (structure_item_id, points_worth) VALUES ($1, 10)`, assignID); err != nil {
		t.Fatal(err)
	}
	qJSON, _ := json.Marshal([]map[string]any{
		{"id": uuid.NewString(), "prompt": "a", "questionType": "multiple_choice", "choices": []string{"1", "2"}, "correctChoiceIndex": 0, "points": 1},
		{"id": uuid.NewString(), "prompt": "b", "questionType": "multiple_choice", "choices": []string{"1", "2"}, "correctChoiceIndex": 0, "points": 1},
	})
	if _, err := pool.Exec(ctx, `INSERT INTO course.module_quizzes (structure_item_id, markdown, questions_json) VALUES ($1, '', $2::jsonb)`, quizID, qJSON); err != nil {
		t.Fatal(err)
	}
	for id, pts := range map[uuid.UUID]float64{assignID: 8, quizID: 1} {
		if _, err := pool.Exec(ctx, `
INSERT INTO course.course_grades (course_id, student_user_id, module_item_id, points_earned) VALUES ($1, $2, $3, $4)`,
			courseID, studentID, id, pts); err != nil {
			t.Fatal(err)
		}
	}

	got, err := PercentForStudent(ctx, pool, courseID, studentID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || math.Abs(*got-75) > 1e-9 {
		t.Fatalf("PercentForStudent = %v, want 75", got)
	}

	// The shared SQL fallback now includes the quiz too: mean of 80% and 50%.
	avg, err := studentprogress.AvgGradePercent(ctx, pool, courseID, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if avg == nil || math.Abs(*avg-65) > 1e-9 {
		t.Fatalf("AvgGradePercent = %v, want 65", avg)
	}
	pct, err := conditionalrelease.ItemScorePercent(ctx, pool, courseID, studentID, quizID)
	if err != nil {
		t.Fatal(err)
	}
	if pct == nil || math.Abs(*pct-50) > 1e-9 {
		t.Fatalf("ItemScorePercent(quiz) = %v, want 50", pct)
	}
}
