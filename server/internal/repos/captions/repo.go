// Package captions provides DB access for auto-captioning jobs (plan 8.4).
package captions

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Status mirrors the storage.caption_status DB enum.
type Status string

const (
	StatusQueued             Status = "queued"
	StatusProcessing         Status = "processing"
	StatusDone               Status = "done"
	StatusFailed             Status = "failed"
	StatusAPIUnavailable     Status = "api_unavailable"
	StatusInstructorReviewed Status = "instructor_reviewed"
)

// Caption is a row from storage.captions.
type Caption struct {
	ID               uuid.UUID
	StorageObjectID  uuid.UUID
	Lang             string
	VTTKey           *string
	TranscriptText   *string
	ConfidenceAvg    *float32
	Backend          string
	Status           Status
	HasLowConfidence bool
	CreatedAt        time.Time
	ReviewedAt       *time.Time
	ReviewedBy       *uuid.UUID
}

// Enqueue inserts a new queued caption job for the given storage object.
func Enqueue(ctx context.Context, pool *pgxpool.Pool, objectID uuid.UUID, backend string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO storage.captions (storage_object_id, backend)
		VALUES ($1, $2)
		RETURNING id`,
		objectID, backend,
	).Scan(&id)
	return id, err
}

// Load fetches a single caption record by ID. Returns nil, nil when not found.
func Load(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Caption, error) {
	return scanCaption(pool.QueryRow(ctx, `
		SELECT id, storage_object_id, lang, vtt_key, transcript_text, confidence_avg,
		       backend, status, has_low_confidence, created_at, reviewed_at, reviewed_by
		FROM storage.captions
		WHERE id = $1`, id))
}

// ListByObjectID returns all caption records for the given storage object.
func ListByObjectID(ctx context.Context, pool *pgxpool.Pool, objectID uuid.UUID) ([]*Caption, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, storage_object_id, lang, vtt_key, transcript_text, confidence_avg,
		       backend, status, has_low_confidence, created_at, reviewed_at, reviewed_by
		FROM storage.captions
		WHERE storage_object_id = $1
		ORDER BY created_at DESC`, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var captions []*Caption
	for rows.Next() {
		c, err := scanCaption(rows)
		if err != nil {
			return nil, err
		}
		captions = append(captions, c)
	}
	return captions, rows.Err()
}

// ClaimNext picks the next queued caption job (FOR UPDATE SKIP LOCKED) and marks it processing.
// Returns nil, nil when no queued jobs are available.
func ClaimNext(ctx context.Context, pool *pgxpool.Pool) (*Caption, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		SELECT id, storage_object_id, lang, vtt_key, transcript_text, confidence_avg,
		       backend, status, has_low_confidence, created_at, reviewed_at, reviewed_by
		FROM storage.captions
		WHERE status = 'queued'
		ORDER BY created_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED`)
	c, err := scanCaption(row)
	if err != nil || c == nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE storage.captions
		SET status = 'processing'
		WHERE id = $1`, c.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	c.Status = StatusProcessing
	return c, nil
}

// MarkDone records successful completion with the VTT key, transcript, and confidence data.
func MarkDone(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, vttKey, lang, transcript string, confidenceAvg float32, hasLowConfidence bool) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.captions
		SET status = 'done', vtt_key = $2, lang = $3, transcript_text = $4,
		    confidence_avg = $5, has_low_confidence = $6
		WHERE id = $1`,
		id, vttKey, lang, transcript, confidenceAvg, hasLowConfidence)
	return err
}

// MarkFailed records a failure with an error message.
func MarkFailed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, apiUnavailable bool) error {
	status := StatusFailed
	if apiUnavailable {
		status = StatusAPIUnavailable
	}
	_, err := pool.Exec(ctx, `
		UPDATE storage.captions
		SET status = $2
		WHERE id = $1`, id, string(status))
	return err
}

// UpdateTranscript updates the transcript text and marks the caption as instructor-reviewed.
// It also regenerates the VTT key reference.
func UpdateTranscript(ctx context.Context, pool *pgxpool.Pool, id, reviewedBy uuid.UUID, transcript, vttKey string) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.captions
		SET transcript_text = $2, vtt_key = $3,
		    status = 'instructor_reviewed', reviewed_at = now(), reviewed_by = $4,
		    has_low_confidence = false
		WHERE id = $1`,
		id, transcript, vttKey, reviewedBy)
	return err
}

// EnqueueForObjectIfNeeded inserts a caption job only when no non-failed job already exists for this object.
func EnqueueForObjectIfNeeded(ctx context.Context, pool *pgxpool.Pool, objectID uuid.UUID, backend string) (uuid.UUID, error) {
	var existing uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT id FROM storage.captions
		WHERE storage_object_id = $1
		  AND status NOT IN ('failed', 'api_unavailable')
		LIMIT 1`, objectID).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	return Enqueue(ctx, pool, objectID, backend)
}

func scanCaption(row pgx.Row) (*Caption, error) {
	var c Caption
	err := row.Scan(
		&c.ID, &c.StorageObjectID, &c.Lang, &c.VTTKey, &c.TranscriptText, &c.ConfidenceAvg,
		&c.Backend, &c.Status, &c.HasLowConfidence, &c.CreatedAt, &c.ReviewedAt, &c.ReviewedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}
