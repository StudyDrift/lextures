// Package lessongenerationjobs persists async AI lesson generation jobs (plan 19.2).
package lessongenerationjobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Row is one lesson_generation_jobs record.
type Row struct {
	ID           uuid.UUID
	InstructorID uuid.UUID
	CourseID     uuid.UUID
	InputParams  json.RawMessage
	Status       string
	Result       json.RawMessage
	ErrorMessage *string
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

// Create inserts a new pending job and returns its id.
func Create(ctx context.Context, pool *pgxpool.Pool, instructorID, courseID uuid.UUID, inputParams json.RawMessage) (uuid.UUID, error) {
	if pool == nil {
		return uuid.Nil, errors.New("lessongenerationjobs: nil pool")
	}
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO jobs.lesson_generation_jobs (instructor_id, course_id, input_params, status)
VALUES ($1, $2, $3, $4)
RETURNING id
`, instructorID, courseID, inputParams, StatusPending).Scan(&id)
	return id, err
}

// GetByID returns a job scoped to instructor and course, or nil when not found.
func GetByID(ctx context.Context, pool *pgxpool.Pool, instructorID, courseID, jobID uuid.UUID) (*Row, error) {
	if pool == nil {
		return nil, errors.New("lessongenerationjobs: nil pool")
	}
	var r Row
	err := pool.QueryRow(ctx, `
SELECT id, instructor_id, course_id, input_params, status, result, error_message, created_at, started_at, completed_at
FROM jobs.lesson_generation_jobs
WHERE id = $1 AND instructor_id = $2 AND course_id = $3
`, jobID, instructorID, courseID).Scan(
		&r.ID, &r.InstructorID, &r.CourseID, &r.InputParams, &r.Status, &r.Result, &r.ErrorMessage,
		&r.CreatedAt, &r.StartedAt, &r.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// MarkProcessing sets status to processing and started_at.
func MarkProcessing(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID) error {
	if pool == nil {
		return errors.New("lessongenerationjobs: nil pool")
	}
	_, err := pool.Exec(ctx, `
UPDATE jobs.lesson_generation_jobs
SET status = $2, started_at = COALESCE(started_at, NOW())
WHERE id = $1
`, jobID, StatusProcessing)
	return err
}

// SaveResult stores the generation result JSON and marks the job completed.
func SaveResult(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, result json.RawMessage) error {
	if pool == nil {
		return errors.New("lessongenerationjobs: nil pool")
	}
	_, err := pool.Exec(ctx, `
UPDATE jobs.lesson_generation_jobs
SET status = $2, result = $3, completed_at = NOW(), error_message = NULL
WHERE id = $1
`, jobID, StatusCompleted, result)
	return err
}

// MarkFailed records a terminal failure message.
func MarkFailed(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, message string) error {
	if pool == nil {
		return errors.New("lessongenerationjobs: nil pool")
	}
	_, err := pool.Exec(ctx, `
UPDATE jobs.lesson_generation_jobs
SET status = $2, error_message = $3, completed_at = NOW()
WHERE id = $1
`, jobID, StatusFailed, message)
	return err
}
