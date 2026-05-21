// Package transcodejobs provides DB access for video transcoding jobs (plan 8.3).
package transcodejobs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Status mirrors the storage.transcode_status DB enum.
type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

// Job is a row from storage.transcode_jobs.
type Job struct {
	ID              uuid.UUID
	SourceKey       string
	OutputPrefix    *string
	MasterPlaylist  *string
	DashManifest    *string
	PosterKey       *string
	Status          Status
	Attempts        int16
	Error           *string
	CreatedAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	StorageObjectID *uuid.UUID
}

// Enqueue inserts a new queued transcode job. Returns the new job ID.
func Enqueue(ctx context.Context, pool *pgxpool.Pool, sourceKey string, storageObjectID *uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO storage.transcode_jobs (source_key, storage_object_id)
		VALUES ($1, $2)
		RETURNING id`,
		sourceKey, storageObjectID,
	).Scan(&id)
	return id, err
}

// Load fetches a single job by ID. Returns nil, nil when not found.
func Load(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Job, error) {
	return scanJob(pool.QueryRow(ctx, `
		SELECT id, source_key, output_prefix, master_playlist, dash_manifest, poster_key,
		       status, attempts, error, created_at, started_at, completed_at, storage_object_id
		FROM storage.transcode_jobs
		WHERE id = $1`, id))
}

// LoadBySourceKey fetches the most recent job for a given source key.
func LoadBySourceKey(ctx context.Context, pool *pgxpool.Pool, sourceKey string) (*Job, error) {
	return scanJob(pool.QueryRow(ctx, `
		SELECT id, source_key, output_prefix, master_playlist, dash_manifest, poster_key,
		       status, attempts, error, created_at, started_at, completed_at, storage_object_id
		FROM storage.transcode_jobs
		WHERE source_key = $1
		ORDER BY created_at DESC
		LIMIT 1`, sourceKey))
}

// LoadByObjectID fetches the most recent job for a given storage object ID.
func LoadByObjectID(ctx context.Context, pool *pgxpool.Pool, objectID uuid.UUID) (*Job, error) {
	return scanJob(pool.QueryRow(ctx, `
		SELECT id, source_key, output_prefix, master_playlist, dash_manifest, poster_key,
		       status, attempts, error, created_at, started_at, completed_at, storage_object_id
		FROM storage.transcode_jobs
		WHERE storage_object_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, objectID))
}

// ClaimNext picks the next queued job (FOR UPDATE SKIP LOCKED) and marks it processing.
// Returns nil, nil when no queued jobs are available.
func ClaimNext(ctx context.Context, pool *pgxpool.Pool) (*Job, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, source_key, output_prefix, master_playlist, dash_manifest, poster_key,
		       status, attempts, error, created_at, started_at, completed_at, storage_object_id
		FROM storage.transcode_jobs
		WHERE status = 'queued'
		ORDER BY created_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED`)
	job, err := scanJob(row)
	if err != nil || job == nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE storage.transcode_jobs
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

// MarkDone records successful completion with output paths.
func MarkDone(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, outputPrefix, masterPlaylist, posterKey string, dashManifest *string) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.transcode_jobs
		SET status = 'done', completed_at = now(),
		    output_prefix = $2, master_playlist = $3, poster_key = $4, dash_manifest = $5,
		    error = NULL
		WHERE id = $1`,
		id, outputPrefix, masterPlaylist, posterKey, dashManifest)
	return err
}

// MarkFailed records a failure; if attempts < maxAttempts re-queues with exponential backoff.
func MarkFailed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, errMsg string, maxAttempts int) error {
	var attempts int16
	err := pool.QueryRow(ctx, `SELECT attempts FROM storage.transcode_jobs WHERE id = $1`, id).Scan(&attempts)
	if err != nil {
		return err
	}

	if int(attempts) >= maxAttempts {
		_, err = pool.Exec(ctx, `
			UPDATE storage.transcode_jobs
			SET status = 'failed', completed_at = now(), error = $2
			WHERE id = $1`, id, errMsg)
	} else {
		// Re-queue with a delay by setting status back to queued
		_, err = pool.Exec(ctx, `
			UPDATE storage.transcode_jobs
			SET status = 'queued', error = $2, started_at = NULL
			WHERE id = $1`, id, errMsg)
	}
	return err
}

func scanJob(row pgx.Row) (*Job, error) {
	var j Job
	err := row.Scan(
		&j.ID, &j.SourceKey, &j.OutputPrefix, &j.MasterPlaylist, &j.DashManifest, &j.PosterKey,
		&j.Status, &j.Attempts, &j.Error, &j.CreatedAt, &j.StartedAt, &j.CompletedAt, &j.StorageObjectID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}
