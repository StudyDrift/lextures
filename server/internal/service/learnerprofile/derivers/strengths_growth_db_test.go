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

func TestStrengthsGrowthDeriver_InsufficientData_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openStrengthsGrowthPool(t, ctx)
	defer pool.Close()

	userID := insertStrengthsGrowthUser(t, ctx, pool)
	defer deleteStrengthsGrowthUser(ctx, pool, userID)

	fix := seedStrengthsGrowthCourse(t, ctx, pool, userID)
	insertConceptState(t, ctx, pool, userID, fix.conceptID, 0.7, 2, time.Now().UTC())

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	deriver := StrengthsGrowthDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}
	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "insufficient_data" {
		t.Fatalf("state=%q want insufficient_data", result.State)
	}
}

func TestStrengthsGrowthDeriver_FullFacet_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx := context.Background()
	pool := openStrengthsGrowthPool(t, ctx)
	defer pool.Close()

	userID := insertStrengthsGrowthUser(t, ctx, pool)
	defer deleteStrengthsGrowthUser(ctx, pool, userID)

	fixedNow := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	recent := fixedNow.AddDate(0, 0, -2)
	decayedSeen := fixedNow.AddDate(0, 0, -40)

	courseA := seedStrengthsGrowthCourse(t, ctx, pool, userID)
	courseB := seedStrengthsGrowthCourse(t, ctx, pool, userID)

	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	linearA := insertStrengthsConcept(t, ctx, pool, courseA.courseID, "linear-a-"+suffix, "Linear equations")
	linearB := insertStrengthsConcept(t, ctx, pool, courseB.courseID, "linear-b-"+suffix, "Linear equations")
	weak := insertStrengthsConcept(t, ctx, pool, courseA.courseID, "unit-conv-"+suffix, "Unit conversions")
	decayed := insertStrengthsConcept(t, ctx, pool, courseA.courseID, "factoring-"+suffix, "Factoring")
	filler := make([]uuid.UUID, 0, 2)
	for i := 0; i < 2; i++ {
		filler = append(filler, insertStrengthsConcept(t, ctx, pool, courseA.courseID, "filler-"+uuid.New().String(), "Filler "+suffix+string(rune('a'+i))))
	}

	insertConceptState(t, ctx, pool, userID, linearA, 0.92, 3, recent)
	insertConceptState(t, ctx, pool, userID, linearB, 0.92, 4, recent)
	insertConceptState(t, ctx, pool, userID, weak, 0.41, 3, recent)
	insertConceptState(t, ctx, pool, userID, decayed, 0.9, 4, decayedSeen)
	for _, id := range filler {
		insertConceptState(t, ctx, pool, userID, id, 0.55, 2, recent)
	}

	misID := insertMisconception(t, ctx, pool, courseA.courseID, weak, "Treats % as additive", "Learners add percentages directly instead of converting to a common base.")
	insertMisconceptionEvents(t, ctx, pool, userID, courseA.courseID, misID, 3)

	deriver := StrengthsGrowthDeriver{Pool: pool, Now: func() time.Time { return fixedNow }}
	result, err := deriver.Derive(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "ok" {
		t.Fatalf("state=%q", result.State)
	}

	var summary StrengthsGrowthSummary
	if err := json.Unmarshal(result.Summary, &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Strengths) == 0 || summary.Strengths[0].Concept != "Linear equations" || summary.Strengths[0].Courses != 2 {
		t.Fatalf("strengths=%+v", summary.Strengths)
	}

	foundGrowth := false
	foundMisconception := false
	for _, item := range summary.Growth {
		if item.Concept == "Unit conversions" {
			foundGrowth = true
		}
		if item.Misconception == "Treats % as additive" {
			foundMisconception = true
		}
	}
	if !foundGrowth || !foundMisconception {
		t.Fatalf("growth=%+v", summary.Growth)
	}

	foundNeedsReview := false
	for _, item := range summary.NeedsReview {
		if item.Concept == "Factoring" && item.LastSeenDays >= 39 {
			foundNeedsReview = true
		}
	}
	if !foundNeedsReview {
		t.Fatalf("needsReview=%+v", summary.NeedsReview)
	}

	svc := learnerprofileservice.New(pool, deriver)
	if err := svc.RecomputeIncremental(ctx, userID, "strengths_growth"); err != nil {
		t.Fatal(err)
	}
	detail, err := svc.GetFacet(ctx, userID, "strengths_growth")
	if err != nil || detail == nil {
		t.Fatalf("get facet: %v %+v", err, detail)
	}
	if detail.Facet.State != "ok" || len(detail.Insights) == 0 {
		t.Fatalf("stored facet=%+v insights=%d", detail.Facet, len(detail.Insights))
	}
}

type strengthsGrowthCourseFixture struct {
	courseID uuid.UUID
	conceptID uuid.UUID
}

func openStrengthsGrowthPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

func insertStrengthsGrowthUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	email := "strengths-growth-" + userID.String() + "@e.invalid"
	if _, err := pool.Exec(ctx, `
INSERT INTO "user".users (id, email, password_hash, display_name)
VALUES ($1, $2, 'hash', 'Strengths Tester')
`, userID, email); err != nil {
		t.Fatal(err)
	}
	return userID
}

func deleteStrengthsGrowthUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) {
	_, _ = pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, userID)
}

func seedStrengthsGrowthCourse(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) strengthsGrowthCourseFixture {
	t.Helper()
	cc := "C-" + strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:6])
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.courses (course_code, title, created_by_user_id)
VALUES ($1, 'Strengths growth test', $2) RETURNING id
`, cc, userID).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active)
VALUES ($1, $2, 'student', TRUE)
`, courseID, userID); err != nil {
		t.Fatal(err)
	}
	conceptID := insertStrengthsConcept(t, ctx, pool, courseID, "seed-"+uuid.New().String(), "Seed concept")
	return strengthsGrowthCourseFixture{courseID: courseID, conceptID: conceptID}
}

func insertStrengthsConcept(t *testing.T, ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, slug, name string) uuid.UUID {
	t.Helper()
	conceptID := uuid.New()
	if _, err := pool.Exec(ctx, `
INSERT INTO course.concepts (id, course_id, name, slug)
VALUES ($1, $2, $3, $4)
`, conceptID, courseID, name, slug); err != nil {
		t.Fatal(err)
	}
	return conceptID
}

func insertConceptState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, conceptID uuid.UUID, mastery float64, attempts int32, lastSeen time.Time) {
	t.Helper()
	reviewAt := lastSeen.AddDate(0, 0, 30)
	if _, err := pool.Exec(ctx, `
INSERT INTO course.learner_concept_states (user_id, concept_id, mastery, attempt_count, last_seen_at, needs_review_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, concept_id) DO UPDATE SET
  mastery = EXCLUDED.mastery,
  attempt_count = EXCLUDED.attempt_count,
  last_seen_at = EXCLUDED.last_seen_at,
  needs_review_at = EXCLUDED.needs_review_at
`, userID, conceptID, mastery, attempts, lastSeen, reviewAt); err != nil {
		t.Fatal(err)
	}
}

func insertMisconception(t *testing.T, ctx context.Context, pool *pgxpool.Pool, courseID, conceptID uuid.UUID, name, description string) uuid.UUID {
	t.Helper()
	misID := uuid.New()
	if _, err := pool.Exec(ctx, `
INSERT INTO course.misconceptions (id, course_id, concept_id, name, description)
VALUES ($1, $2, $3, $4, $5)
`, misID, courseID, conceptID, name, description); err != nil {
		t.Fatal(err)
	}
	return misID
}

func insertMisconceptionEvents(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID, courseID, misconceptionID uuid.UUID, count int) {
	t.Helper()
	var moduleID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 0, 'module', 'Mod', NULL, TRUE) RETURNING id
`, courseID).Scan(&moduleID); err != nil {
		t.Fatal(err)
	}
	var quizID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, parent_id, published)
VALUES ($1, 1, 'quiz', 'Quiz', $2, TRUE) RETURNING id
`, courseID, moduleID).Scan(&quizID); err != nil {
		t.Fatal(err)
	}
	questionID := uuid.New()
	if _, err := pool.Exec(ctx, `
INSERT INTO course.questions (id, course_id, stem, question_type, options, status)
VALUES ($1, $2, 'Q?', 'mc_single', '[]'::jsonb, 'active')
`, questionID, courseID); err != nil {
		t.Fatal(err)
	}
	var attemptID uuid.UUID
	if err := pool.QueryRow(ctx, `
INSERT INTO course.quiz_attempts (course_id, structure_item_id, student_user_id, attempt_number, status)
VALUES ($1, $2, $3, 1, 'submitted') RETURNING id
`, courseID, quizID, userID).Scan(&attemptID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < count; i++ {
		if _, err := pool.Exec(ctx, `
INSERT INTO course.misconception_events (course_id, user_id, attempt_id, question_id, misconception_id)
VALUES ($1, $2, $3, $4, $5)
`, courseID, userID, attemptID, questionID, misconceptionID); err != nil {
			t.Fatal(err)
		}
	}
}