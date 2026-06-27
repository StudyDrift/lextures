package background

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
)

func workerTestPool(t *testing.T) *pgxpool.Pool {
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
	if _, err := pool.Exec(ctx, `TRUNCATE jobs.queue, jobs.dead_letters`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func newTestWorker(pool *pgxpool.Pool, reg *Registry) *jobWorker {
	return &jobWorker{pool: pool, registry: reg, concurrency: 4, visibility: time.Minute}
}

func TestWorker_ProcessesJobToCompletion(t *testing.T) {
	pool := workerTestPool(t)
	ctx := context.Background()
	now := time.Now().UTC()

	var ran atomic.Int32
	reg := NewRegistry()
	reg.Register("test.ok", HandlerFunc(func(_ context.Context, payload json.RawMessage) error {
		var p struct {
			N int `json:"n"`
		}
		_ = json.Unmarshal(payload, &p)
		if p.N == 42 {
			ran.Add(1)
		}
		return nil
	}))

	id, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     "test.ok",
		Payload:     map[string]any{"n": 42},
		ScheduledAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	newTestWorker(pool, reg).tick(ctx)

	if ran.Load() != 1 {
		t.Fatalf("handler should have run once, ran=%d", ran.Load())
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM jobs.queue WHERE id = $1`, id).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "completed" {
		t.Fatalf("want completed, got %s", status)
	}
}

func TestWorker_FailingJobDeadLetters(t *testing.T) {
	pool := workerTestPool(t)
	ctx := context.Background()
	now := time.Now().UTC()

	reg := NewRegistry()
	reg.Register("test.fail", HandlerFunc(func(_ context.Context, _ json.RawMessage) error {
		return errors.New("always fails")
	}))

	id, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     "test.fail",
		MaxAttempts: 1,
		ScheduledAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	newTestWorker(pool, reg).tick(ctx)

	var dlCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs.dead_letters WHERE id = $1`, id).Scan(&dlCount); err != nil {
		t.Fatal(err)
	}
	if dlCount != 1 {
		t.Fatalf("failing job with max_attempts=1 should dead-letter, dl=%d", dlCount)
	}
}

func TestWorker_UnknownTypeIsHandledGracefully(t *testing.T) {
	pool := workerTestPool(t)
	ctx := context.Background()
	now := time.Now().UTC()

	id, err := jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     "no.handler",
		MaxAttempts: 1,
		ScheduledAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Empty registry: the worker must not panic and should fail the job.
	newTestWorker(pool, NewRegistry()).tick(ctx)

	var inQueue int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs.queue WHERE id = $1 AND status = 'completed'`, id).Scan(&inQueue); err != nil {
		t.Fatal(err)
	}
	if inQueue != 0 {
		t.Fatal("unknown-type job must not be marked completed")
	}
}
