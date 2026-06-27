// Package jobqueue is the durable Postgres-backed store for the generic
// background job queue (plan 17.3). It provides enqueue, SKIP LOCKED claim with
// a visibility timeout, retry with exponential backoff, dead-letter handling,
// and the queries powering the admin jobs UI.
package jobqueue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultMaxAttempts is used when an enqueue does not specify one.
const DefaultMaxAttempts = 5

// DefaultVisibilityTimeout is how long a claimed ("running") job may run before
// a worker is presumed dead and the row becomes reclaimable (plan 17.3 AC-3).
const DefaultVisibilityTimeout = 10 * time.Minute

// EnqueueParams describes a job to insert.
type EnqueueParams struct {
	JobType     string
	Payload     any    // marshalled to JSON; nil becomes {}
	Priority    int    // 1 (highest) .. 10 (lowest); 0 defaults to 5
	MaxAttempts int    // 0 defaults to DefaultMaxAttempts
	UniqueKey   string // optional dedup key; empty disables dedup
	ScheduledAt time.Time
}

// Claimed is a job leased to a worker for execution.
type Claimed struct {
	ID          uuid.UUID
	JobType     string
	Payload     json.RawMessage
	Attempts    int
	MaxAttempts int
	Priority    int
	UniqueKey   *string
}

// Enqueue inserts a job and returns its id. When UniqueKey is set and an
// in-flight job with that key already exists, the existing job's id is returned
// and no new row is created (plan 17.3 FR-8).
func Enqueue(ctx context.Context, pool *pgxpool.Pool, p EnqueueParams) (uuid.UUID, error) {
	priority := p.Priority
	if priority <= 0 {
		priority = 5
	}
	maxAttempts := p.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}
	scheduledAt := p.ScheduledAt
	if scheduledAt.IsZero() {
		scheduledAt = time.Now().UTC()
	}
	raw, err := marshalPayload(p.Payload)
	if err != nil {
		return uuid.Nil, err
	}
	var uniqueKey *string
	if p.UniqueKey != "" {
		uniqueKey = &p.UniqueKey
	}

	var id uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO jobs.queue (job_type, payload, priority, max_attempts, unique_key, scheduled_at)
VALUES ($1, $2::jsonb, $3, $4, $5, $6)
ON CONFLICT (unique_key) WHERE unique_key IS NOT NULL AND status IN ('pending','running','failed')
DO NOTHING
RETURNING id
`, p.JobType, raw, priority, maxAttempts, uniqueKey, scheduledAt).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// Deduped against an existing in-flight job; return its id.
		if uniqueKey == nil {
			return uuid.Nil, err
		}
		qerr := pool.QueryRow(ctx, `
SELECT id FROM jobs.queue
WHERE unique_key = $1 AND status IN ('pending','running','failed')
ORDER BY created_at
LIMIT 1
`, *uniqueKey).Scan(&id)
		return id, qerr
	}
	return id, err
}

// Claim leases up to limit ready jobs to this worker using SELECT ... FOR UPDATE
// SKIP LOCKED so concurrent workers never double-process a row (plan 17.3 AC-4).
// Rows whose visibility timeout has elapsed while "running" are reclaimed, which
// recovers jobs orphaned by a worker crash (plan 17.3 AC-3). Claiming a job
// increments its attempt counter.
func Claim(ctx context.Context, pool *pgxpool.Pool, limit int, now time.Time, visibility time.Duration) ([]Claimed, error) {
	if limit <= 0 {
		limit = 1
	}
	if visibility <= 0 {
		visibility = DefaultVisibilityTimeout
	}
	staleBefore := now.Add(-visibility)
	rows, err := pool.Query(ctx, `
WITH claimed AS (
    SELECT id
    FROM jobs.queue
    WHERE (status IN ('pending','failed') AND scheduled_at <= $1)
       OR (status = 'running' AND started_at < $2)
    ORDER BY priority ASC, scheduled_at ASC
    LIMIT $3
    FOR UPDATE SKIP LOCKED
)
UPDATE jobs.queue q
SET status = 'running', started_at = $1, attempts = q.attempts + 1
FROM claimed
WHERE q.id = claimed.id
RETURNING q.id, q.job_type, q.payload, q.attempts, q.max_attempts, q.priority, q.unique_key
`, now, staleBefore, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Claimed
	for rows.Next() {
		var c Claimed
		if err := rows.Scan(&c.ID, &c.JobType, &c.Payload, &c.Attempts, &c.MaxAttempts, &c.Priority, &c.UniqueKey); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Complete marks a running job done.
func Complete(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, now time.Time) error {
	_, err := pool.Exec(ctx, `
UPDATE jobs.queue
SET status = 'completed', completed_at = $2, error_log = NULL
WHERE id = $1
`, id, now)
	return err
}

// Fail records a failed attempt. If the job still has attempts remaining it is
// re-scheduled with exponential backoff (plan 17.3 FR-4); otherwise it is moved
// to the dead-letter table. Returns true when the job was dead-lettered.
func Fail(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, now time.Time, errMsg string) (deadLettered bool, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		jobType     string
		payload     json.RawMessage
		priority    int
		attempts    int
		maxAttempts int
		uniqueKey   *string
	)
	err = tx.QueryRow(ctx, `
SELECT job_type, payload, priority, attempts, max_attempts, unique_key
FROM jobs.queue WHERE id = $1 FOR UPDATE
`, id).Scan(&jobType, &payload, &priority, &attempts, &maxAttempts, &uniqueKey)
	if err != nil {
		return false, err
	}

	if attempts >= maxAttempts {
		if _, err = tx.Exec(ctx, `
INSERT INTO jobs.dead_letters (id, job_type, payload, priority, unique_key, attempts, error_log)
VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7)
`, id, jobType, payload, priority, uniqueKey, attempts, truncateErr(errMsg)); err != nil {
			return false, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM jobs.queue WHERE id = $1`, id); err != nil {
			return false, err
		}
		if err = tx.Commit(ctx); err != nil {
			return false, err
		}
		return true, nil
	}

	next := NextRetryAt(now, attempts)
	if _, err = tx.Exec(ctx, `
UPDATE jobs.queue
SET status = 'failed', scheduled_at = $2, error_log = $3, started_at = NULL
WHERE id = $1
`, id, next, truncateErr(errMsg)); err != nil {
		return false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return false, nil
}

// Stats summarises queue health for the admin UI and metrics (plan 17.3 §10).
type Stats struct {
	Pending     int            `json:"pending"`
	Running     int            `json:"running"`
	Failed      int            `json:"failed"`
	DeadLetters int            `json:"deadLetters"`
	Depth       int            `json:"depth"` // pending + failed + running
	ByType      map[string]int `json:"byType"`
}

// GetStats returns counts by status, queue depth by job type, and dead-letter
// depth.
func GetStats(ctx context.Context, pool *pgxpool.Pool) (Stats, error) {
	s := Stats{ByType: map[string]int{}}
	rows, err := pool.Query(ctx, `SELECT status, count(*) FROM jobs.queue GROUP BY status`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			rows.Close()
			return s, err
		}
		switch status {
		case "pending":
			s.Pending += n
		case "running":
			s.Running += n
		case "failed":
			s.Failed += n
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return s, err
	}
	s.Depth = s.Pending + s.Running + s.Failed

	typeRows, err := pool.Query(ctx, `
SELECT job_type, count(*) FROM jobs.queue
WHERE status IN ('pending','running','failed')
GROUP BY job_type ORDER BY count(*) DESC
`)
	if err != nil {
		return s, err
	}
	for typeRows.Next() {
		var jt string
		var n int
		if err := typeRows.Scan(&jt, &n); err != nil {
			typeRows.Close()
			return s, err
		}
		s.ByType[jt] = n
	}
	typeRows.Close()
	if err := typeRows.Err(); err != nil {
		return s, err
	}

	if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs.dead_letters WHERE NOT redriven`).Scan(&s.DeadLetters); err != nil {
		return s, err
	}
	return s, nil
}

// JobRow is a queue row for admin listing.
type JobRow struct {
	ID          uuid.UUID  `json:"id"`
	JobType     string     `json:"jobType"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"maxAttempts"`
	ScheduledAt time.Time  `json:"scheduledAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	ErrorLog    *string    `json:"errorLog,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ListJobs returns recent queue rows, newest first.
func ListJobs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]JobRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, job_type, status, priority, attempts, max_attempts, scheduled_at, started_at, error_log, created_at
FROM jobs.queue
ORDER BY created_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]JobRow, 0, limit)
	for rows.Next() {
		var j JobRow
		if err := rows.Scan(&j.ID, &j.JobType, &j.Status, &j.Priority, &j.Attempts, &j.MaxAttempts,
			&j.ScheduledAt, &j.StartedAt, &j.ErrorLog, &j.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// DeadLetter is a row in the dead-letter table.
type DeadLetter struct {
	ID       uuid.UUID `json:"id"`
	JobType  string    `json:"jobType"`
	Attempts int       `json:"attempts"`
	ErrorLog *string   `json:"errorLog,omitempty"`
	FailedAt time.Time `json:"failedAt"`
	Redriven bool      `json:"redriven"`
}

// ListDeadLetters returns dead-letter rows, newest failure first.
func ListDeadLetters(ctx context.Context, pool *pgxpool.Pool, limit int) ([]DeadLetter, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, job_type, attempts, error_log, failed_at, redriven
FROM jobs.dead_letters
ORDER BY failed_at DESC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DeadLetter, 0, limit)
	for rows.Next() {
		var d DeadLetter
		if err := rows.Scan(&d.ID, &d.JobType, &d.Attempts, &d.ErrorLog, &d.FailedAt, &d.Redriven); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ErrNotFound is returned when a redrive or cancel targets a missing/ineligible job.
var ErrNotFound = errors.New("jobqueue: job not found")

// Redrive re-enqueues a dead-letter job with a fresh attempt count (plan 17.3
// AC-5). The dead-letter row is marked redriven and a new pending queue row is
// created. Returns the new job id.
func Redrive(ctx context.Context, pool *pgxpool.Pool, deadLetterID uuid.UUID, now time.Time) (uuid.UUID, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		jobType  string
		payload  json.RawMessage
		priority int
	)
	err = tx.QueryRow(ctx, `
SELECT job_type, payload, priority FROM jobs.dead_letters
WHERE id = $1 AND NOT redriven FOR UPDATE
`, deadLetterID).Scan(&jobType, &payload, &priority)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}

	var newID uuid.UUID
	if err = tx.QueryRow(ctx, `
INSERT INTO jobs.queue (job_type, payload, priority, scheduled_at)
VALUES ($1, $2::jsonb, $3, $4)
RETURNING id
`, jobType, payload, priority, now).Scan(&newID); err != nil {
		return uuid.Nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE jobs.dead_letters SET redriven = true WHERE id = $1`, deadLetterID); err != nil {
		return uuid.Nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return newID, nil
}

// Cancel deletes a pending job (plan 17.3 §9 DELETE /admin/jobs/{id}). Running,
// completed, or already-removed jobs cannot be cancelled.
func Cancel(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	tag, err := pool.Exec(ctx, `DELETE FROM jobs.queue WHERE id = $1 AND status IN ('pending','failed')`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func marshalPayload(v any) ([]byte, error) {
	if v == nil {
		return []byte(`{}`), nil
	}
	if raw, ok := v.(json.RawMessage); ok {
		if len(raw) == 0 {
			return []byte(`{}`), nil
		}
		return raw, nil
	}
	return json.Marshal(v)
}

// truncateErr bounds stored error text so a pathological error message cannot
// bloat the row (plan 17.3 NFR security: error logs kept small, no PII dumps).
func truncateErr(s string) string {
	const max = 4000
	if len(s) > max {
		return s[:max]
	}
	return s
}
