// Package avscanjobs provides DB access for AV scan jobs (plan 8.6).
package avscanjobs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Status mirrors storage.av_scan_job_status.
type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

// Job is a row from storage.av_scan_jobs.
type Job struct {
	ID              uuid.UUID
	StorageObjectID uuid.UUID
	Status          Status
	Attempts        int16
	Error           *string
	CreatedAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
}

// Enqueue inserts a queued AV scan job for the given storage object.
func Enqueue(ctx context.Context, pool *pgxpool.Pool, storageObjectID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO storage.av_scan_jobs (storage_object_id)
		SELECT $1
		WHERE NOT EXISTS (
		  SELECT 1 FROM storage.av_scan_jobs
		  WHERE storage_object_id = $1 AND status IN ('queued', 'processing')
		)
		RETURNING id`, storageObjectID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already queued — return existing job id if any.
		err = pool.QueryRow(ctx, `
			SELECT id FROM storage.av_scan_jobs
			WHERE storage_object_id = $1 AND status IN ('queued', 'processing')
			ORDER BY created_at DESC LIMIT 1`, storageObjectID).Scan(&id)
	}
	return id, err
}

// ClaimNext picks the next queued job (FOR UPDATE SKIP LOCKED) and marks it processing.
func ClaimNext(ctx context.Context, pool *pgxpool.Pool) (*Job, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, storage_object_id, status, attempts, error, created_at, started_at, completed_at
		FROM storage.av_scan_jobs
		WHERE status = 'queued'
		ORDER BY created_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED`)
	job, err := scanJob(row)
	if err != nil || job == nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE storage.av_scan_jobs
		SET status = 'processing', started_at = now(), attempts = attempts + 1
		WHERE id = $1`, job.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	job.Status = StatusProcessing
	return job, nil
}

// MarkDone records successful completion.
func MarkDone(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.av_scan_jobs
		SET status = 'done', completed_at = now(), error = NULL
		WHERE id = $1`, id)
	return err
}

// MarkFailed records failure; re-queues if attempts remain below maxAttempts.
func MarkFailed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, errMsg string, maxAttempts int) error {
	var attempts int16
	err := pool.QueryRow(ctx, `SELECT attempts FROM storage.av_scan_jobs WHERE id = $1`, id).Scan(&attempts)
	if err != nil {
		return err
	}
	if int(attempts) >= maxAttempts {
		_, err = pool.Exec(ctx, `
			UPDATE storage.av_scan_jobs
			SET status = 'failed', completed_at = now(), error = $2
			WHERE id = $1`, id, errMsg)
	} else {
		_, err = pool.Exec(ctx, `
			UPDATE storage.av_scan_jobs
			SET status = 'queued', error = $2, started_at = NULL
			WHERE id = $1`, id, errMsg)
	}
	return err
}

// BulkEnqueuePending queues scan jobs for legacy pending objects.
func BulkEnqueuePending(ctx context.Context, pool *pgxpool.Pool, objectIDs []uuid.UUID) (int, error) {
	n := 0
	for _, oid := range objectIDs {
		if _, err := Enqueue(ctx, pool, oid); err == nil {
			n++
		}
	}
	return n, nil
}

func scanJob(row pgx.Row) (*Job, error) {
	var j Job
	var status string
	err := row.Scan(
		&j.ID, &j.StorageObjectID, &status, &j.Attempts, &j.Error,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	j.Status = Status(status)
	return &j, nil
}
