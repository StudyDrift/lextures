package scheduler

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// acquireLock atomically takes the distributed lock for jobName, returning true
// only when no unexpired holder exists (plan 17.4 FR-2, AC-2). Concurrent
// instances racing for the same trigger contend on the job_name primary key, so
// exactly one upsert wins.
func acquireLock(ctx context.Context, pool *pgxpool.Pool, jobName, instanceID string, now time.Time, ttl time.Duration) (bool, error) {
	expires := now.Add(ttl)
	tag, err := pool.Exec(ctx, `
INSERT INTO jobs.schedule_locks (job_name, locked_by, locked_at, expires_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (job_name) DO UPDATE
SET locked_by = EXCLUDED.locked_by, locked_at = EXCLUDED.locked_at, expires_at = EXCLUDED.expires_at
WHERE jobs.schedule_locks.expires_at < $3
`, jobName, instanceID, now, expires)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// releaseLock expires the lock immediately so the row does not block a later
// manual trigger; the schedule-history row already prevents a re-fire this tick.
func releaseLock(ctx context.Context, pool *pgxpool.Pool, jobName, instanceID string) error {
	_, err := pool.Exec(ctx, `
DELETE FROM jobs.schedule_locks WHERE job_name = $1 AND locked_by = $2
`, jobName, instanceID)
	return err
}

// recordHistory writes one schedule-trigger row linking to the enqueued queue
// job (plan 17.4 FR-5).
func recordHistory(ctx context.Context, pool *pgxpool.Pool, jobName string, triggeredAt time.Time, jobID uuid.UUID, status, notes string) error {
	var jobIDArg any
	if jobID != uuid.Nil {
		jobIDArg = jobID
	}
	var notesArg any
	if notes != "" {
		notesArg = notes
	}
	_, err := pool.Exec(ctx, `
INSERT INTO jobs.schedule_history (job_name, triggered_at, job_id, status, notes)
VALUES ($1, $2, $3, $4, $5)
`, jobName, triggeredAt, jobIDArg, status, notesArg)
	return err
}

// lastTriggered returns the most recent trigger time for a job, or zero when the
// job has never fired.
func lastTriggered(ctx context.Context, pool *pgxpool.Pool, jobName string) (time.Time, error) {
	var t time.Time
	err := pool.QueryRow(ctx, `
SELECT triggered_at FROM jobs.schedule_history
WHERE job_name = $1 ORDER BY triggered_at DESC LIMIT 1
`, jobName).Scan(&t)
	if err != nil {
		if isNoRows(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return t, nil
}

// overrides returns the persisted enable/disable overrides keyed by job name
// (plan 17.4 FR-6). Missing entries mean "use the code default".
func overrides(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT job_name, enabled FROM jobs.schedule_overrides`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		var enabled bool
		if err := rows.Scan(&name, &enabled); err != nil {
			return nil, err
		}
		out[name] = enabled
	}
	return out, rows.Err()
}

// SetEnabled upserts an admin enable/disable override for a job (plan 17.4 FR-6,
// API POST .../enable and .../disable).
func SetEnabled(ctx context.Context, pool *pgxpool.Pool, jobName string, enabled bool, now time.Time) error {
	_, err := pool.Exec(ctx, `
INSERT INTO jobs.schedule_overrides (job_name, enabled, updated_at)
VALUES ($1, $2, $3)
ON CONFLICT (job_name) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at
`, jobName, enabled, now)
	return err
}

// HistoryRow is one schedule-history entry resolved against the queue for the
// admin UI (plan 17.4 §10).
type HistoryRow struct {
	ID          int64      `json:"id"`
	JobName     string     `json:"jobName"`
	TriggeredAt time.Time  `json:"triggeredAt"`
	JobID       *uuid.UUID `json:"jobId,omitempty"`
	// Status is the live queue/dead-letter status of the enqueued job, falling
	// back to the history status when the queue row is gone (plan 17.4 AC-4).
	Status   string  `json:"status"`
	ErrorLog *string `json:"errorLog,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

// ListHistory returns the most recent trigger rows for a job, newest first,
// joined to the live job status so a failed run is visible (plan 17.4 AC-4).
func ListHistory(ctx context.Context, pool *pgxpool.Pool, jobName string, limit int) ([]HistoryRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
SELECT h.id, h.job_name, h.triggered_at, h.job_id,
       COALESCE(q.status, CASE WHEN dl.id IS NOT NULL THEN 'dead_letter' END, h.status) AS status,
       COALESCE(q.error_log, dl.error_log),
       h.notes
FROM jobs.schedule_history h
LEFT JOIN jobs.queue q ON q.id = h.job_id
LEFT JOIN jobs.dead_letters dl ON dl.id = h.job_id
WHERE h.job_name = $1
ORDER BY h.triggered_at DESC
LIMIT $2
`, jobName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]HistoryRow, 0, limit)
	for rows.Next() {
		var h HistoryRow
		if err := rows.Scan(&h.ID, &h.JobName, &h.TriggeredAt, &h.JobID, &h.Status, &h.ErrorLog, &h.Notes); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
