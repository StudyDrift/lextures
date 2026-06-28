package scheduler

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
)

// testJobName is unique enough not to collide with the built-in schedules so the
// tests can run against a shared dev database without touching real rows.
const testJobName = "test_sched_job"
const testJobType = "test.scheduled.noop"

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
	cleanTestRows(t, pool)
	t.Cleanup(func() { cleanTestRows(t, pool); pool.Close() })
	return pool
}

func cleanTestRows(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	exec := func(q string, arg string) {
		if _, err := pool.Exec(ctx, q, arg); err != nil {
			t.Fatalf("clean: %v", err)
		}
	}
	exec(`DELETE FROM jobs.queue WHERE job_type = $1`, testJobType)
	exec(`DELETE FROM jobs.schedule_history WHERE job_name = $1`, testJobName)
	exec(`DELETE FROM jobs.schedule_locks WHERE job_name = $1`, testJobName)
	exec(`DELETE FROM jobs.schedule_overrides WHERE job_name = $1`, testJobName)
}

// newTestScheduler builds a Scheduler with a single hourly test job so behaviour
// is deterministic and isolated from the built-in schedules.
func newTestScheduler(pool *pgxpool.Pool, instanceID string, startedAt time.Time) *Scheduler {
	return &Scheduler{
		pool:       pool,
		instanceID: instanceID,
		startedAt:  startedAt,
		ttl:        lockTTL,
		jobs: []ScheduledJob{{
			Name:           testJobName,
			Spec:           "0 * * * *",
			JobType:        testJobType,
			Description:    "test job",
			DefaultEnabled: true,
			schedule:       MustParse("0 * * * *"),
		}},
	}
}

func countQueue(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM jobs.queue WHERE job_type = $1`, testJobType).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// AC-5: a schedule whose last run predates a missed interval fires on the tick.
func TestTickFiresMissedJob(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	// startedAt long ago so the hourly job is overdue immediately.
	s := newTestScheduler(pool, "inst-a", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	now := time.Now().UTC().Truncate(time.Minute)

	s.Tick(ctx, now)
	if got := countQueue(t, pool); got != 1 {
		t.Fatalf("expected 1 queued job after first tick, got %d", got)
	}

	// A second tick in the same hour must not re-fire (history advances the anchor).
	s.Tick(ctx, now.Add(time.Second))
	if got := countQueue(t, pool); got != 1 {
		t.Fatalf("expected still 1 queued job, got %d", got)
	}
}

// AC-2: two instances firing the same trigger enqueue exactly one job.
func TestTwoInstancesEnqueueOnce(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	a := newTestScheduler(pool, "inst-a", start)
	b := newTestScheduler(pool, "inst-b", start)
	now := time.Now().UTC().Truncate(time.Minute)

	a.Tick(ctx, now)
	b.Tick(ctx, now)

	if got := countQueue(t, pool); got != 1 {
		t.Fatalf("expected exactly 1 queued job across two instances, got %d", got)
	}
	var historyRows int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM jobs.schedule_history WHERE job_name = $1`, testJobName).Scan(&historyRows); err != nil {
		t.Fatal(err)
	}
	if historyRows != 1 {
		t.Fatalf("expected 1 history row, got %d", historyRows)
	}
}

func TestDisabledJobDoesNotFire(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := newTestScheduler(pool, "inst-a", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	now := time.Now().UTC().Truncate(time.Minute)

	if err := SetEnabled(ctx, pool, testJobName, false, now); err != nil {
		t.Fatal(err)
	}
	s.Tick(ctx, now)
	if got := countQueue(t, pool); got != 0 {
		t.Fatalf("disabled job should not enqueue, got %d", got)
	}

	// Re-enable and verify it fires.
	if err := SetEnabled(ctx, pool, testJobName, true, now); err != nil {
		t.Fatal(err)
	}
	s.Tick(ctx, now)
	if got := countQueue(t, pool); got != 1 {
		t.Fatalf("re-enabled job should enqueue, got %d", got)
	}
}

func TestManualTrigger(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := newTestScheduler(pool, "inst-a", time.Now().UTC())

	id, err := s.Trigger(ctx, testJobName, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if id.String() == "" {
		t.Fatal("expected a job id")
	}
	if got := countQueue(t, pool); got != 1 {
		t.Fatalf("manual trigger should enqueue 1 job, got %d", got)
	}

	if _, err := s.Trigger(ctx, "nonexistent", time.Now().UTC()); err != ErrUnknownJob {
		t.Fatalf("expected ErrUnknownJob, got %v", err)
	}
}

func TestLockContention(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	now := time.Now().UTC()

	ok, err := acquireLock(ctx, pool, testJobName, "inst-a", now, time.Minute)
	if err != nil || !ok {
		t.Fatalf("first acquire should succeed: ok=%v err=%v", ok, err)
	}
	ok2, err := acquireLock(ctx, pool, testJobName, "inst-b", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if ok2 {
		t.Fatal("second acquire should fail while lock is held")
	}
	// After expiry another instance may take it.
	ok3, err := acquireLock(ctx, pool, testJobName, "inst-b", now.Add(2*time.Minute), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !ok3 {
		t.Fatal("acquire after expiry should succeed")
	}
}
