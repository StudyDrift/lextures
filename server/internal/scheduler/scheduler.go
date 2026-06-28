package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/jobqueue"
)

// TickInterval is how often the scheduler evaluates its schedules. A minute is
// fine because schedules have minute resolution; the per-tick overhead is a
// handful of cheap queries (plan 17.4 NFR performance).
const TickInterval = time.Minute

// lockTTL bounds how long a held lock survives. It is short relative to every
// schedule interval (the most frequent built-in is hourly) so a crashed holder
// cannot starve a schedule, while still covering the enqueue + history write
// against a racing instance (plan 17.4 NFR reliability/scalability).
const lockTTL = 5 * time.Minute

// Scheduler evaluates the configured schedules on a tick loop and enqueues due
// jobs onto the durable queue (plan 17.4). It is safe to run on every instance;
// the distributed lock ensures exactly one fires each trigger.
type Scheduler struct {
	pool       *pgxpool.Pool
	jobs       []ScheduledJob
	instanceID string
	startedAt  time.Time
	ttl        time.Duration
}

// New builds a Scheduler over the built-in job list. instanceID identifies this
// process in the lock table; a hostname+pid default is used when empty.
func New(pool *pgxpool.Pool, instanceID string) *Scheduler {
	if instanceID == "" {
		host, _ := os.Hostname()
		instanceID = fmt.Sprintf("%s-%d", host, os.Getpid())
	}
	return &Scheduler{
		pool:       pool,
		jobs:       BuiltinJobs(),
		instanceID: instanceID,
		startedAt:  time.Now().UTC(),
		ttl:        lockTTL,
	}
}

// Jobs returns the configured scheduled jobs (for the admin API).
func (s *Scheduler) Jobs() []ScheduledJob { return s.jobs }

// job returns the configured job by name.
func (s *Scheduler) job(name string) (ScheduledJob, bool) {
	for _, j := range s.jobs {
		if j.Name == name {
			return j, true
		}
	}
	return ScheduledJob{}, false
}

// Start runs the tick loop until ctx is cancelled. It returns immediately; the
// loop runs in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		t := time.NewTicker(TickInterval)
		defer t.Stop()
		// Evaluate once on startup so a schedule missed while the app was down
		// re-fires without waiting a full tick (plan 17.4 AC-5).
		s.Tick(ctx, time.Now().UTC())
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.Tick(ctx, time.Now().UTC())
			}
		}
	}()
	slog.Info("scheduler started", "instance", s.instanceID, "jobs", len(s.jobs))
}

// Tick evaluates every schedule once. For each due, enabled job it acquires the
// distributed lock, enqueues a queue job, and records history. Errors on one job
// never abort the others.
func (s *Scheduler) Tick(ctx context.Context, now time.Time) {
	now = now.UTC()
	ov, err := overrides(ctx, s.pool)
	if err != nil {
		slog.Warn("scheduler.overrides", "err", err)
		ov = map[string]bool{}
	}
	for _, j := range s.jobs {
		enabled := j.DefaultEnabled
		if v, ok := ov[j.Name]; ok {
			enabled = v
		}
		if !enabled {
			continue
		}
		if err := s.evaluate(ctx, j, now); err != nil {
			slog.Warn("scheduler.evaluate", "job", j.Name, "err", err)
		}
	}
}

// evaluate fires one job if it is due. The lock is taken before the due check is
// committed (history write) so two instances cannot both enqueue the same
// trigger; the loser simply finds the lock held and skips (plan 17.4 AC-2).
func (s *Scheduler) evaluate(ctx context.Context, j ScheduledJob, now time.Time) error {
	last, err := lastTriggered(ctx, s.pool, j.Name)
	if err != nil {
		return err
	}
	anchor := last
	if anchor.IsZero() {
		anchor = s.startedAt
	}
	if !j.Schedule().IsDue(anchor, now) {
		return nil
	}

	acquired, err := acquireLock(ctx, s.pool, j.Name, s.instanceID, now, s.ttl)
	if err != nil {
		return err
	}
	if !acquired {
		return nil // another instance is handling this trigger
	}
	defer func() { _ = releaseLock(ctx, s.pool, j.Name, s.instanceID) }()

	// Re-check under the lock: another instance may have fired between our due
	// check and lock acquisition.
	last2, err := lastTriggered(ctx, s.pool, j.Name)
	if err != nil {
		return err
	}
	anchor2 := last2
	if anchor2.IsZero() {
		anchor2 = s.startedAt
	}
	if !j.Schedule().IsDue(anchor2, now) {
		return nil
	}

	return s.fire(ctx, j, now)
}

// fire enqueues the job onto the durable queue and records the trigger. The
// unique_key dedups against any still-in-flight prior run of the same schedule
// (plan 17.4 FR-3). The history triggered_at is floored to the scheduled minute
// so the next IsDue check advances deterministically.
func (s *Scheduler) fire(ctx context.Context, j ScheduledJob, now time.Time) error {
	triggeredAt := now.Truncate(time.Minute)
	jobID, err := jobqueue.Enqueue(ctx, s.pool, jobqueue.EnqueueParams{
		JobType:   j.JobType,
		Priority:  5,
		UniqueKey: fmt.Sprintf("schedule:%s:%d", j.Name, triggeredAt.Unix()),
	})
	if err != nil {
		// Record the failed trigger so the miss is visible in history.
		_ = recordHistory(ctx, s.pool, j.Name, triggeredAt, uuid.Nil, "enqueue_failed", err.Error())
		return err
	}
	if err := recordHistory(ctx, s.pool, j.Name, triggeredAt, jobID, "triggered", ""); err != nil {
		return err
	}
	slog.Info("scheduler.fired", "job", j.Name, "job_id", jobID, "triggered_at", triggeredAt)
	return nil
}

// JobSummary describes a scheduled job's configuration and recent run state for
// the admin UI (plan 17.4 §10).
type JobSummary struct {
	Name        string     `json:"name"`
	Spec        string     `json:"spec"`
	JobType     string     `json:"jobType"`
	Description string     `json:"description"`
	Enabled     bool       `json:"enabled"`
	LastRun     *time.Time `json:"lastRun,omitempty"`
	LastStatus  string     `json:"lastStatus,omitempty"`
	NextRun     *time.Time `json:"nextRun,omitempty"`
}

// Summary returns every configured job with its enabled state, last run, last
// status and next scheduled run (plan 17.4 §9 GET /admin/scheduler).
func (s *Scheduler) Summary(ctx context.Context, now time.Time) ([]JobSummary, error) {
	now = now.UTC()
	ov, err := overrides(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	out := make([]JobSummary, 0, len(s.jobs))
	for _, j := range s.jobs {
		enabled := j.DefaultEnabled
		if v, ok := ov[j.Name]; ok {
			enabled = v
		}
		sum := JobSummary{
			Name:        j.Name,
			Spec:        j.Spec,
			JobType:     j.JobType,
			Description: j.Description,
			Enabled:     enabled,
		}
		if enabled {
			next := j.Schedule().Next(now)
			if !next.IsZero() {
				sum.NextRun = &next
			}
		}
		hist, err := ListHistory(ctx, s.pool, j.Name, 1)
		if err != nil {
			return nil, err
		}
		if len(hist) > 0 {
			t := hist[0].TriggeredAt
			sum.LastRun = &t
			sum.LastStatus = hist[0].Status
		}
		out = append(out, sum)
	}
	return out, nil
}

// Trigger manually fires a job now, bypassing the schedule (plan 17.4 §9 POST
// .../trigger). It still enqueues onto the queue and records history. Returns the
// enqueued job id.
func (s *Scheduler) Trigger(ctx context.Context, name string, now time.Time) (uuid.UUID, error) {
	j, ok := s.job(name)
	if !ok {
		return uuid.Nil, ErrUnknownJob
	}
	now = now.UTC()
	jobID, err := jobqueue.Enqueue(ctx, s.pool, jobqueue.EnqueueParams{
		JobType:   j.JobType,
		Priority:  5,
		UniqueKey: fmt.Sprintf("schedule:%s:manual:%d", j.Name, now.UnixNano()),
	})
	if err != nil {
		return uuid.Nil, err
	}
	if err := recordHistory(ctx, s.pool, j.Name, now, jobID, "triggered", "manual"); err != nil {
		return uuid.Nil, err
	}
	slog.Info("scheduler.manual_trigger", "job", j.Name, "job_id", jobID)
	return jobID, nil
}

// ErrUnknownJob is returned when an admin action names a job that is not
// configured.
var ErrUnknownJob = fmt.Errorf("scheduler: unknown job")
