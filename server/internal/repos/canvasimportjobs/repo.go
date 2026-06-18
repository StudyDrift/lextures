// Package canvasimportjobs persists queued Canvas LMS import jobs.
package canvasimportjobs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Status mirrors jobs.canvas_import_jobs.status.
type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// Include mirrors the Canvas import include flags stored as JSONB.
type Include struct {
	Modules     bool `json:"modules"`
	Assignments bool `json:"assignments"`
	Quizzes     bool `json:"quizzes"`
	Enrollments bool `json:"enrollments"`
	Grades      bool `json:"grades"`
	Settings    bool `json:"settings"`
	Files       bool `json:"files"`
}

// Job is a row from jobs.canvas_import_jobs (access token is not stored).
type Job struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	CourseCode     string
	Status         Status
	Mode           string
	CanvasBaseURL  string
	CanvasCourseID string
	Include        Include
	LastProgress   *string
	ErrorMessage   *string
	CourseTitle    *string
	Attempts       int16
	MaxAttempts    int16
	CreatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

// QueueMessage is the RabbitMQ payload (includes the Canvas token, not stored in Postgres).
type QueueMessage struct {
	JobID          uuid.UUID `json:"jobId"`
	UserID         uuid.UUID `json:"userId"`
	CourseCode     string    `json:"courseCode"`
	Mode           string    `json:"mode"`
	CanvasBaseURL  string    `json:"canvasBaseUrl"`
	CanvasCourseID string    `json:"canvasCourseId"`
	AccessToken    string    `json:"accessToken"`
	Include        Include   `json:"include"`
}

// Insert creates a queued job row and returns its ID.
func Insert(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	courseCode, mode, canvasBaseURL, canvasCourseID string,
	include Include,
) (uuid.UUID, error) {
	includeJSON, err := json.Marshal(include)
	if err != nil {
		return uuid.UUID{}, err
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO jobs.canvas_import_jobs
			(user_id, course_code, mode, canvas_base_url, canvas_course_id, include)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		RETURNING id`,
		userID, courseCode, mode, canvasBaseURL, canvasCourseID, includeJSON,
	).Scan(&id)
	return id, err
}

// LatestCompletedForCourse returns the most recent successful Canvas import for a course, if any.
func LatestCompletedForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*Job, error) {
	return scanJob(pool.QueryRow(ctx, `
		SELECT id, user_id, course_code, status, mode, canvas_base_url, canvas_course_id,
		       include, last_progress, error_message, course_title, attempts, max_attempts,
		       created_at, started_at, completed_at
		FROM jobs.canvas_import_jobs
		WHERE course_code = $1 AND status = 'completed'
		ORDER BY completed_at DESC NULLS LAST, created_at DESC
		LIMIT 1`, strings.TrimSpace(courseCode)))
}

// LatestLinkedForCourse returns the newest Canvas import job for a course (any status).
// Used to detect Canvas-linked courses even when the latest import is still running or failed.
func LatestLinkedForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*Job, error) {
	return scanJob(pool.QueryRow(ctx, `
		SELECT id, user_id, course_code, status, mode, canvas_base_url, canvas_course_id,
		       include, last_progress, error_message, course_title, attempts, max_attempts,
		       created_at, started_at, completed_at
		FROM jobs.canvas_import_jobs
		WHERE course_code = $1
		ORDER BY created_at DESC
		LIMIT 1`, strings.TrimSpace(courseCode)))
}

// Load fetches a job by ID. Returns nil, nil when not found.
func Load(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Job, error) {
	return scanJob(pool.QueryRow(ctx, `
		SELECT id, user_id, course_code, status, mode, canvas_base_url, canvas_course_id,
		       include, last_progress, error_message, course_title, attempts, max_attempts,
		       created_at, started_at, completed_at
		FROM jobs.canvas_import_jobs
		WHERE id = $1`, id))
}

// MarkProcessing sets status to processing and increments attempts.
func MarkProcessing(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE jobs.canvas_import_jobs
		SET status = 'processing', started_at = COALESCE(started_at, now()), attempts = attempts + 1
		WHERE id = $1 AND status IN ('queued', 'processing')`, id)
	return err
}

// UpdateProgress stores the latest progress line for reconnecting clients.
func UpdateProgress(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, message string) error {
	_, err := pool.Exec(ctx, `
		UPDATE jobs.canvas_import_jobs SET last_progress = $2 WHERE id = $1`, id, message)
	return err
}

// MarkCompleted marks a job successful and stores the imported course title when known.
func MarkCompleted(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, courseTitle string) error {
	_, err := pool.Exec(ctx, `
		UPDATE jobs.canvas_import_jobs
		SET status = 'completed', completed_at = now(), course_title = NULLIF($2, ''), error_message = NULL
		WHERE id = $1`, id, courseTitle)
	return err
}

// MarkFailed records a terminal error; requeues when attempts remain under max_attempts.
func MarkFailed(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, errMsg string, maxAttempts int16) error {
	var attempts int16
	if err := pool.QueryRow(ctx, `SELECT attempts FROM jobs.canvas_import_jobs WHERE id = $1`, id).Scan(&attempts); err != nil {
		return err
	}
	if attempts < maxAttempts {
		_, err := pool.Exec(ctx, `
			UPDATE jobs.canvas_import_jobs
			SET status = 'queued', error_message = $2, started_at = NULL
			WHERE id = $1`, id, errMsg)
		return err
	}
	_, err := pool.Exec(ctx, `
		UPDATE jobs.canvas_import_jobs
		SET status = 'failed', completed_at = now(), error_message = $2
		WHERE id = $1`, id, errMsg)
	return err
}

func scanJob(row pgx.Row) (*Job, error) {
	var j Job
	var includeRaw []byte
	var lastProgress, errMsg, courseTitle *string
	err := row.Scan(
		&j.ID, &j.UserID, &j.CourseCode, &j.Status, &j.Mode, &j.CanvasBaseURL, &j.CanvasCourseID,
		&includeRaw, &lastProgress, &errMsg, &courseTitle, &j.Attempts, &j.MaxAttempts,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(includeRaw) > 0 {
		_ = json.Unmarshal(includeRaw, &j.Include)
	}
	j.LastProgress = lastProgress
	j.ErrorMessage = errMsg
	j.CourseTitle = courseTitle
	return &j, nil
}
