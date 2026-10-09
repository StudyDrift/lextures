package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// Submitting an auto-gradable quiz writes the score to course_grades without staff clicking
// "Save scores" in the quiz grader.
func TestQuizSubmit_WritesGradebookCell_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	email := fmt.Sprintf("quiz-sub-%s@test.invalid", uuid.NewString())
	ph, err := auth.HashPassword("longpassword0longpassword0")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	row, err := user.InsertUser(ctx, pool, email, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	studentID, _ := uuid.Parse(row.ID)

	cc := fmt.Sprintf("C-%06X", uuid.New().ID()%0xFFFFFF)
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id) VALUES ($1, 'Quiz submit sync', $2) RETURNING id
`, cc, studentID).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO course.course_enrollments (course_id, user_id, role) VALUES ($1, $2, 'student')`, courseID, studentID); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	var moduleID, quizID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 0, 'module', 'Mod', NULL, TRUE) RETURNING id`, courseID).Scan(&moduleID); err != nil {
		t.Fatalf("module: %v", err)
	}
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 1, 'quiz', 'QA Quiz', $2, TRUE) RETURNING id`, courseID, moduleID).Scan(&quizID); err != nil {
		t.Fatalf("quiz: %v", err)
	}
	q1, q2 := uuid.NewString(), uuid.NewString()
	qJSON, _ := json.Marshal([]map[string]any{
		{"id": q1, "prompt": "2+2?", "questionType": "multiple_choice", "choices": []string{"4", "5"}, "correctChoiceIndex": 0, "points": 1},
		{"id": q2, "prompt": "3+3?", "questionType": "multiple_choice", "choices": []string{"6", "7"}, "correctChoiceIndex": 0, "points": 1},
	})
	if _, err := pool.Exec(ctx, `INSERT INTO course.module_quizzes (structure_item_id, markdown, questions_json) VALUES ($1, '', $2::jsonb)`, quizID, qJSON); err != nil {
		t.Fatalf("module quiz: %v", err)
	}
	att, err := quizattempts.InsertAttempt(ctx, pool, quizattempts.InsertAttemptParams{
		CourseID: courseID, StructureItemID: quizID, StudentUserID: studentID, AttemptNumber: 1,
	})
	if err != nil {
		t.Fatalf("attempt: %v", err)
	}

	signer := auth.NewJWTSignerWithPool("01234567890123456789012345678901", pool)
	tok, err := signer.Sign(ctx, row.ID, email, "", "", nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	h := NewHandler(Deps{Pool: pool, JWTSigner: signer, Config: config.Config{}})

	// One right, one wrong: 1/2.
	body, _ := json.Marshal(map[string]any{
		"attemptId": att.ID,
		"responses": []map[string]any{
			{"questionId": q1, "selectedChoiceIndex": 0},
			{"questionId": q2, "selectedChoiceIndex": 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/courses/%s/quizzes/%s/submit", cc, quizID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("submit: %d %s", w.Code, w.Body.String())
	}

	var pts float64
	if err := pool.QueryRow(ctx, `
SELECT points_earned::float8 FROM course.course_grades WHERE student_user_id = $1 AND module_item_id = $2
`, studentID, quizID).Scan(&pts); err != nil {
		t.Fatalf("gradebook cell not written on submit: %v", err)
	}
	if pts != 1 {
		t.Fatalf("points_earned = %v want 1", pts)
	}
}
