package quizautosubmit

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
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/questionbank"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/learnerstate"
)

type masteryQuizFixture struct {
	pool      *pgxpool.Pool
	courseID  uuid.UUID
	quizID    uuid.UUID
	conceptID uuid.UUID
	questionID string
	cfg       config.Config
}

func TestSweepExpiredAttempts_MasteryMatchesManualSubmit_Pg(t *testing.T) {
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

	fix, manualUser, sweepUser := seedMasteryQuizFixture(t, ctx, pool)
	now := time.Now().UTC()
	deadline := now.Add(-time.Minute)

	manualAttempt := insertTimedQuizAttempt(t, ctx, pool, fix, manualUser, deadline)
	sweepAttempt := insertTimedQuizAttempt(t, ctx, pool, fix, sweepUser, deadline)
	insertCorrectResponse(t, ctx, pool, manualAttempt, fix.questionID)
	insertCorrectResponse(t, ctx, pool, sweepAttempt, fix.questionID)

	manualMastery := applyManualMastery(t, ctx, pool, fix, manualAttempt)
	finalizeAttemptSubmitted(t, ctx, pool, manualAttempt, now)
	swept, err := SweepExpiredAttempts(ctx, pool, fix.cfg, now, 10)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if swept != 1 {
		t.Fatalf("swept %d attempts want 1", swept)
	}
	sweepMastery := readConceptMastery(t, ctx, pool, sweepUser, fix.conceptID)
	if manualMastery != sweepMastery {
		t.Fatalf("manual mastery %v != sweep mastery %v", manualMastery, sweepMastery)
	}
}

func seedMasteryQuizFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (masteryQuizFixture, uuid.UUID, uuid.UUID) {
	t.Helper()
	ts := time.Now().Format("20060102150405")
	manualUser := insertTestStudent(t, ctx, pool, "manual-"+ts+"@example.com")
	sweepUser := insertTestStudent(t, ctx, pool, "sweep-"+ts+"@example.com")

	cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6])
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id)
VALUES ($1, 'Mastery sweep test', $2) RETURNING id
`, cc, manualUser).Scan(&courseID); err != nil {
		t.Fatalf("course: %v", err)
	}

	conceptID := uuid.New()
	slug := "linear-" + strings.ReplaceAll(conceptID.String(), "-", "")
	if _, err := pool.Exec(ctx, `
INSERT INTO course.concepts (id, course_id, name, slug)
VALUES ($1, $2, 'Linear equations', $3)
`, conceptID, courseID, slug); err != nil {
		t.Fatalf("concept: %v", err)
	}

	var moduleID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 0, 'module', 'Mod', NULL, TRUE) RETURNING id
`, courseID).Scan(&moduleID); err != nil {
		t.Fatalf("module: %v", err)
	}

	questionID := uuid.New().String()
	correct := uint(0)
	questions := []map[string]any{
		{
			"id":                 questionID,
			"prompt":             "2+2?",
			"questionType":       "multiple_choice",
			"choices":            []string{"4", "5"},
			"correctChoiceIndex": correct,
			"points":             1,
			"conceptIds":         []string{conceptID.String()},
		},
	}
	qJSON, err := json.Marshal(questions)
	if err != nil {
		t.Fatalf("marshal questions: %v", err)
	}

	var quizID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 1, 'quiz', 'Timed quiz', $2, TRUE) RETURNING id
`, courseID, moduleID).Scan(&quizID); err != nil {
		t.Fatalf("quiz item: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.module_quizzes (structure_item_id, markdown, questions_json)
VALUES ($1, '', $2::jsonb)
`, quizID, qJSON); err != nil {
		t.Fatalf("module quiz: %v", err)
	}

	return masteryQuizFixture{
		pool:       pool,
		courseID:   courseID,
		quizID:     quizID,
		conceptID:  conceptID,
		questionID: questionID,
		cfg: config.Config{
			AdaptiveLearnerModelEnabled: true,
			LearnerModelEMAAlpha:         0.3,
		},
	}, manualUser, sweepUser
}

func insertTestStudent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	ph, err := auth.HashPassword("password1230password1230")
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.InsertUser(ctx, pool, email, ph, nil)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	uid, err := uuid.Parse(u.ID)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	return uid
}

func insertTimedQuizAttempt(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fix masteryQuizFixture, studentID uuid.UUID, deadline time.Time) uuid.UUID {
	t.Helper()
	att, err := quizattempts.InsertAttempt(ctx, pool, quizattempts.InsertAttemptParams{
		CourseID:        fix.courseID,
		StructureItemID: fix.quizID,
		StudentUserID:   studentID,
		AttemptNumber:   1,
		DeadlineAt:      &deadline,
	})
	if err != nil {
		t.Fatalf("insert attempt: %v", err)
	}
	return att.ID
}

func insertCorrectResponse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID, questionID string) {
	t.Helper()
	correct := true
	resp, _ := json.Marshal(map[string]any{"selectedChoiceIndex": 0})
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := quizattempts.ReplaceResponses(ctx, tx, attemptID, []quizattempts.ResponseRow{
		{
			QuestionIndex: 0,
			QuestionID:    questionID,
			QuestionType:  "multiple_choice",
			ResponseJSON:  resp,
			IsCorrect:     &correct,
			PointsAwarded: 1,
			MaxPoints:     1,
		},
	}); err != nil {
		t.Fatalf("responses: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit responses: %v", err)
	}
}

func applyManualMastery(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fix masteryQuizFixture, attemptID uuid.UUID) float64 {
	t.Helper()
	meta, err := course.GetCourseQuizMeta(ctx, pool, fix.courseID)
	if err != nil || meta == nil {
		t.Fatalf("course meta: %v", err)
	}
	row, err := coursemodulequizzes.GetForCourseItem(ctx, pool, fix.courseID, fix.quizID)
	if err != nil || row == nil {
		t.Fatalf("quiz row: %v", err)
	}
	questions, _, err := questionbank.ResolveDeliveryQuestionsForGet(
		ctx, pool, fix.courseID, fix.quizID, meta.QuestionBankEnabled, row.Questions, &attemptID, false,
	)
	if err != nil {
		t.Fatalf("resolve questions: %v", err)
	}
	responses, err := quizattempts.ListResponses(ctx, pool, attemptID)
	if err != nil {
		t.Fatalf("list responses: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var studentID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT student_user_id FROM course.quiz_attempts WHERE id = $1`, attemptID).Scan(&studentID); err != nil {
		t.Fatalf("student: %v", err)
	}
	if err := learnerstate.ApplyMasteryFromSavedResponses(ctx, pool, tx, learnerstate.ApplyMasteryParams{
		CourseID:                      fix.courseID,
		UserID:                        studentID,
		AttemptID:                     attemptID,
		Questions:                     questions,
		Responses:                     responses,
		HintScaffoldingEnabled:        meta.HintScaffoldingEnabled,
		MisconceptionDetectionEnabled: meta.MisconceptionDetectionEnabled,
		LearnerModelEnabled:           fix.cfg.AdaptiveLearnerModelEnabled,
		EMAAlpha:                      fix.cfg.LearnerModelEMAAlpha,
	}); err != nil {
		t.Fatalf("manual mastery: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return readConceptMastery(t, ctx, pool, studentID, fix.conceptID)
}

func finalizeAttemptSubmitted(t *testing.T, ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID, submittedAt time.Time) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin finalize: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ok, err := quizattempts.FinalizeAttemptSubmitted(ctx, tx, quizattempts.FinalizeSubmitParams{
		AttemptID:      attemptID,
		SubmittedAt:    submittedAt,
		PointsEarned:   1,
		PointsPossible: 1,
		ScorePercent:   100,
	})
	if err != nil || !ok {
		t.Fatalf("finalize manual attempt: ok=%v err=%v", ok, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit finalize: %v", err)
	}
}

func readConceptMastery(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, conceptID uuid.UUID) float64 {
	t.Helper()
	var mastery float64
	err := pool.QueryRow(ctx, `
SELECT (mastery)::float8 FROM course.learner_concept_states WHERE user_id = $1 AND concept_id = $2
`, userID, conceptID).Scan(&mastery)
	if err != nil {
		t.Fatalf("read mastery: %v", err)
	}
	return mastery
}