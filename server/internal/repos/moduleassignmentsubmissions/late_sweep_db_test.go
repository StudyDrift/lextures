package moduleassignmentsubmissions_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	subrepo "github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
)

func testPool(t *testing.T) *pgxpool.Pool {
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
	return pool
}

// seedSubmission creates a user, course, due-dated assignment item and a
// submission for it, returning the submission id. dueOffset is relative to now;
// submitOffset is relative to the due date. Everything is cleaned up via t.Cleanup.
func seedSubmission(t *testing.T, pool *pgxpool.Pool, now time.Time, dueOffset, submitOffset time.Duration) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()
	// course_code must match ^C-[A-Z0-9]{6}$.
	code := "C-" + strings.ToUpper(strings.ReplaceAll(suffix, "-", ""))[:6]

	var userID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO "user".users (email, password_hash) VALUES ($1, 'x') RETURNING id`,
		"late-"+suffix[:8]+"@example.test").Scan(&userID); err != nil {
		t.Fatal(err)
	}
	var courseID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO course.courses (course_code, title) VALUES ($1, 'Test') RETURNING id`,
		code).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	due := now.Add(dueOffset)
	var itemID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO course.course_structure_items (course_id, sort_order, kind, title, due_at)
		 VALUES ($1, 1, 'module', 'Assignment', $2) RETURNING id`,
		courseID, due).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	var subID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO course.module_assignment_submissions (course_id, module_item_id, submitted_by, submitted_at)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		courseID, itemID, userID, due.Add(submitOffset)).Scan(&subID); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM course.courses WHERE id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM "user".users WHERE id = $1`, userID)
	})
	return subID
}

func isLate(t *testing.T, pool *pgxpool.Pool, subID uuid.UUID) bool {
	t.Helper()
	var late bool
	if err := pool.QueryRow(context.Background(),
		`SELECT is_late FROM course.module_assignment_submissions WHERE id = $1`, subID).Scan(&late); err != nil {
		t.Fatal(err)
	}
	return late
}

// AC-1: a submission turned in after the due date is marked late by the sweep.
func TestMarkOverdueLate(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Due 2h ago, submitted 1h after due (i.e. late) — should be flagged.
	lateSub := seedSubmission(t, pool, now, -2*time.Hour, time.Hour)
	// Due 2h ago, submitted 1h before due (on time) — should not be flagged.
	onTimeSub := seedSubmission(t, pool, now, -2*time.Hour, -time.Hour)

	n, err := subrepo.MarkOverdueLate(ctx, pool, now)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 row marked, got %d", n)
	}
	if !isLate(t, pool, lateSub) {
		t.Error("late submission should be flagged is_late")
	}
	if isLate(t, pool, onTimeSub) {
		t.Error("on-time submission should not be flagged is_late")
	}

	// Idempotent: a second run does not re-mark the already-late row.
	if !isLate(t, pool, lateSub) {
		t.Fatal("precondition")
	}
	if _, err := subrepo.MarkOverdueLate(ctx, pool, now); err != nil {
		t.Fatal(err)
	}
}
