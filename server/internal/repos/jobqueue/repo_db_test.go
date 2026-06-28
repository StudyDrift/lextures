package jobqueue_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
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
	reset(t, pool)
	return pool
}

func reset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE jobs.queue, jobs.dead_letters, jobs.schedule_history`); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueClaimComplete(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	id, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     "test.noop",
		Payload:     map[string]any{"hello": "world"},
		ScheduledAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	claimed, err := jobqueue.Claim(ctx, pool, 10, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].ID != id {
		t.Fatalf("want 1 claimed job %s, got %+v", id, claimed)
	}
	if claimed[0].Attempts != 1 {
		t.Fatalf("claim should increment attempts to 1, got %d", claimed[0].Attempts)
	}
	if string(claimed[0].Payload) != `{"hello": "world"}` {
		t.Fatalf("payload roundtrip mismatch: %s", claimed[0].Payload)
	}

	// A second claim must not re-pick the running job.
	again, err := jobqueue.Claim(ctx, pool, 10, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("running job should not be re-claimed, got %d", len(again))
	}

	if err := jobqueue.Complete(ctx, pool, id, now); err != nil {
		t.Fatal(err)
	}
	stats, err := jobqueue.GetStats(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Depth != 0 {
		t.Fatalf("completed job should not count toward depth, got %+v", stats)
	}
}

func TestDedupByUniqueKey(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()

	id1, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{JobType: "t", UniqueKey: "k1"})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{JobType: "t", UniqueKey: "k1"})
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("dedup should return existing id: %s vs %s", id1, id2)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs.queue WHERE unique_key = 'k1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("want 1 row for unique key, got %d", count)
	}
}

func TestRetryThenDeadLetterAndRedrive(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	id, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     "test.flaky",
		MaxAttempts: 1, // dead-letter on first failure
		ScheduledAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := jobqueue.Claim(ctx, pool, 1, now, time.Minute)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v %+v", err, claimed)
	}
	dead, err := jobqueue.Fail(ctx, pool, id, now, "smtp unreachable")
	if err != nil {
		t.Fatal(err)
	}
	if !dead {
		t.Fatal("expected dead-letter after exhausting attempts")
	}
	var qcount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs.queue WHERE id = $1`, id).Scan(&qcount); err != nil {
		t.Fatal(err)
	}
	if qcount != 0 {
		t.Fatal("dead-lettered job should be removed from queue")
	}
	dls, err := jobqueue.ListDeadLetters(ctx, pool, 10)
	if err != nil || len(dls) != 1 {
		t.Fatalf("dead letters: %v %+v", err, dls)
	}

	newID, err := jobqueue.Redrive(ctx, pool, dls[0].ID, now)
	if err != nil {
		t.Fatal(err)
	}
	reclaimed, err := jobqueue.Claim(ctx, pool, 1, now, time.Minute)
	if err != nil || len(reclaimed) != 1 || reclaimed[0].ID != newID {
		t.Fatalf("redriven job should be claimable: %v %+v", err, reclaimed)
	}
	// Redriving again must be a no-op (already redriven).
	if _, err := jobqueue.Redrive(ctx, pool, dls[0].ID, now); err != jobqueue.ErrNotFound {
		t.Fatalf("re-redrive want ErrNotFound, got %v", err)
	}
}

func TestRetryReschedulesWithBackoff(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	id, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{JobType: "t", MaxAttempts: 3, ScheduledAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobqueue.Claim(ctx, pool, 1, now, time.Minute); err != nil {
		t.Fatal(err)
	}
	dead, err := jobqueue.Fail(ctx, pool, id, now, "transient")
	if err != nil {
		t.Fatal(err)
	}
	if dead {
		t.Fatal("should not dead-letter with attempts remaining")
	}
	// Not eligible immediately (scheduled 1 min out), but eligible after backoff.
	if c, _ := jobqueue.Claim(ctx, pool, 1, now, time.Minute); len(c) != 0 {
		t.Fatalf("failed job should wait for backoff, claimed %d", len(c))
	}
	later := now.Add(2 * time.Minute)
	if c, _ := jobqueue.Claim(ctx, pool, 1, later, time.Minute); len(c) != 1 {
		t.Fatalf("failed job should be eligible after backoff, claimed %d", len(c))
	}
}

func TestScheduledJobNotClaimedEarly(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     "test.delayed",
		ScheduledAt: now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if c, _ := jobqueue.Claim(ctx, pool, 5, now, time.Minute); len(c) != 0 {
		t.Fatalf("delayed job claimed early: %d", len(c))
	}
	if c, _ := jobqueue.Claim(ctx, pool, 5, now.Add(11*time.Minute), time.Minute); len(c) != 1 {
		t.Fatalf("delayed job not claimed after schedule: %d", len(c))
	}
}

func TestVisibilityTimeoutReclaim(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	id, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{JobType: "t", ScheduledAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := jobqueue.Claim(ctx, pool, 1, now, 10*time.Minute); len(c) != 1 {
		t.Fatalf("initial claim failed: %d", len(c))
	}
	// Worker "crashes". 11 minutes later, the row is reclaimable.
	future := now.Add(11 * time.Minute)
	c, err := jobqueue.Claim(ctx, pool, 1, future, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != 1 || c[0].ID != id {
		t.Fatalf("visibility timeout did not reclaim job: %+v", c)
	}
	if c[0].Attempts != 2 {
		t.Fatalf("reclaim should re-increment attempts, got %d", c[0].Attempts)
	}
}

func TestConcurrentClaimNoDoubleProcessing(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	const n = 50
	for i := 0; i < n; i++ {
		if _, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{JobType: "t", ScheduledAt: now}); err != nil {
			t.Fatal(err)
		}
	}

	var (
		mu      sync.Mutex
		seen    = map[uuid.UUID]int{}
		wg      sync.WaitGroup
		workers = 8
	)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				claimed, err := jobqueue.Claim(ctx, pool, 3, now, time.Minute)
				if err != nil || len(claimed) == 0 {
					return
				}
				mu.Lock()
				for _, c := range claimed {
					seen[c.ID]++
				}
				mu.Unlock()
				for _, c := range claimed {
					_ = jobqueue.Complete(ctx, pool, c.ID, now)
				}
			}
		}()
	}
	wg.Wait()

	if len(seen) != n {
		t.Fatalf("want %d distinct jobs processed, got %d", n, len(seen))
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("job %s processed %d times (SKIP LOCKED violated)", id, count)
		}
	}
}

func TestCancelPendingOnly(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	id, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{JobType: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if err := jobqueue.Cancel(ctx, pool, id); err != nil {
		t.Fatalf("cancel pending: %v", err)
	}
	if err := jobqueue.Cancel(ctx, pool, id); err != jobqueue.ErrNotFound {
		t.Fatalf("cancel missing want ErrNotFound, got %v", err)
	}

	// Running jobs cannot be cancelled.
	id2, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{JobType: "t", ScheduledAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobqueue.Claim(ctx, pool, 1, now, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := jobqueue.Cancel(ctx, pool, id2); err != jobqueue.ErrNotFound {
		t.Fatalf("cancel running want ErrNotFound, got %v", err)
	}
}
