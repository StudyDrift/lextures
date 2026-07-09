package background

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/jobqueue"
)

// Handler executes one job type. Implementations must be idempotent: a job may
// be delivered more than once (worker crash after side effect, visibility
// timeout reclaim), so handlers should tolerate re-execution (plan 17.3 FR-1).
type Handler interface {
	Execute(ctx context.Context, payload json.RawMessage) error
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(ctx context.Context, payload json.RawMessage) error

// Execute implements Handler.
func (f HandlerFunc) Execute(ctx context.Context, payload json.RawMessage) error {
	return f(ctx, payload)
}

// Registry maps a job type to its handler. Adding a new job type means
// registering one handler here; nothing else in the worker changes
// (plan 17.3 NFR maintainability).
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

// Register adds (or replaces) the handler for a job type.
func (r *Registry) Register(jobType string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[jobType] = h
}

// handler returns the handler for a job type.
func (r *Registry) handler(jobType string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[jobType]
	return h, ok
}

// Types returns the registered job types (for diagnostics/tests).
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		out = append(out, t)
	}
	return out
}

// jobWorker polls the durable queue and dispatches claimed jobs to handlers.
type jobWorker struct {
	pool        *pgxpool.Pool
	registry    *Registry
	concurrency int
	visibility  time.Duration
}

// StartJobQueueWorker launches the generic background-job worker when the
// feature flag is on (plan 17.3 FR-5, rollout flag background_jobs_enabled). It
// returns immediately; the worker runs until ctx is cancelled. Returns the
// registry so callers can register job types (already populated with the
// built-in types).
func StartJobQueueWorker(ctx context.Context, pool *pgxpool.Pool, cfgSrc ConfigSource) *Registry {
	registry := NewRegistry()
	RegisterBuiltinJobs(registry, pool, cfgSrc)
	cfg := cfgSrc.Config()
	if pool == nil || !cfg.BackgroundJobsEnabled {
		return registry
	}
	concurrency := cfg.BackgroundJobsConcurrency
	if concurrency < 1 {
		concurrency = 4
	}
	w := &jobWorker{
		pool:        pool,
		registry:    registry,
		concurrency: concurrency,
		visibility:  jobqueue.DefaultVisibilityTimeout,
	}
	go runEvery(ctx, time.Second, func() {
		w.tick(context.Background())
	})
	slog.Info("background job queue worker started", "concurrency", concurrency)
	return registry
}

// tick claims a batch of ready jobs and runs them, bounded by concurrency.
func (w *jobWorker) tick(ctx context.Context) {
	claimed, err := jobqueue.Claim(ctx, w.pool, w.concurrency, time.Now().UTC(), w.visibility)
	if err != nil {
		slog.Warn("job_queue.claim", "err", err)
		return
	}
	if len(claimed) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, job := range claimed {
		wg.Add(1)
		go func(job jobqueue.Claimed) {
			defer wg.Done()
			w.run(ctx, job)
		}(job)
	}
	wg.Wait()
}

// run executes one claimed job and records the outcome.
func (w *jobWorker) run(ctx context.Context, job jobqueue.Claimed) {
	now := time.Now().UTC()
	h, ok := w.registry.handler(job.JobType)
	if !ok {
		// Unknown type: fail it so it retries/dead-letters rather than spinning.
		dead, ferr := jobqueue.Fail(ctx, w.pool, job.ID, now, "no handler registered for job type "+job.JobType)
		if ferr != nil {
			slog.Warn("job_queue.fail", "job_id", job.ID, "err", ferr)
		}
		slog.Warn("job_queue.unknown_type", "job_id", job.ID, "job_type", job.JobType, "dead_lettered", dead)
		return
	}

	err := safeExecute(ctx, h, job.Payload)
	if err != nil {
		dead, ferr := jobqueue.Fail(ctx, w.pool, job.ID, time.Now().UTC(), err.Error())
		if ferr != nil {
			slog.Warn("job_queue.fail", "job_id", job.ID, "err", ferr)
			return
		}
		slog.Warn("job_queue.job_failed",
			"job_id", job.ID, "job_type", job.JobType,
			"attempt", job.Attempts, "max_attempts", job.MaxAttempts,
			"dead_lettered", dead, "err", err)
		return
	}
	if err := jobqueue.Complete(ctx, w.pool, job.ID, time.Now().UTC()); err != nil {
		slog.Warn("job_queue.complete", "job_id", job.ID, "err", err)
	}
}

// safeExecute runs a handler, converting a panic into an error so one bad job
// cannot crash the worker.
func safeExecute(ctx context.Context, h Handler, payload json.RawMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("job handler panic: %v", r)
		}
	}()
	return h.Execute(ctx, payload)
}
