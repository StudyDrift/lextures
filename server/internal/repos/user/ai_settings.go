// user.user_ai_settings — per-user OpenRouter model defaults (mirrors server/src/repos/user_ai_settings.rs).
package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Defaults when no row exists (parity with Rust repos/user_ai_settings.rs).
const (
	DefaultImageModelID                = "black-forest-labs/flux.2-flex"
	DefaultCourseSetupModelID          = "arcee-ai/trinity-mini:free"
	DefaultNotebookFlashcardsModelID   = "arcee-ai/trinity-mini:free"
	DefaultVibeActivityModelID         = "arcee-ai/trinity-mini:free"
	DefaultGraderAgentModelID          = "arcee-ai/trinity-mini:free"
)

// GetImageModelID returns the user's image model, or the global default.
func GetImageModelID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	if pool == nil {
		return "", errors.New("db pool is nil")
	}
	var s string
	err := pool.QueryRow(ctx, `SELECT image_model_id FROM "user".user_ai_settings WHERE user_id = $1`, userID).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultImageModelID, nil
	}
	if err != nil {
		return "", err
	}
	return s, nil
}

// GetCourseSetupModelID returns the user's text model for course setup, or the default.
func GetCourseSetupModelID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	if pool == nil {
		return "", errors.New("db pool is nil")
	}
	var s string
	err := pool.QueryRow(ctx, `SELECT course_setup_model_id FROM "user".user_ai_settings WHERE user_id = $1`, userID).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultCourseSetupModelID, nil
	}
	if err != nil {
		return "", err
	}
	return s, nil
}

// GetNotebookFlashcardsModelID returns the model to use for AI flashcard generation.
func GetNotebookFlashcardsModelID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	if pool == nil {
		return "", errors.New("db pool is nil")
	}
	var s string
	err := pool.QueryRow(ctx, `SELECT notebook_flashcards_model_id FROM "user".user_ai_settings WHERE user_id = $1`, userID).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultNotebookFlashcardsModelID, nil
	}
	if err != nil {
		return "", err
	}
	return s, nil
}

// GetVibeActivityModelID returns the model to use for AI vibe activity generation.
func GetVibeActivityModelID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	if pool == nil {
		return "", errors.New("db pool is nil")
	}
	var s string
	err := pool.QueryRow(ctx, `SELECT vibe_activity_model_id FROM "user".user_ai_settings WHERE user_id = $1`, userID).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultVibeActivityModelID, nil
	}
	if err != nil {
		return "", err
	}
	return s, nil
}

// GetGraderAgentModelID returns the model to use for the grading agent.
func GetGraderAgentModelID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	if pool == nil {
		return "", errors.New("db pool is nil")
	}
	var s string
	err := pool.QueryRow(ctx, `SELECT grader_agent_model_id FROM "user".user_ai_settings WHERE user_id = $1`, userID).Scan(&s)
	if errors.Is(err, pgx.ErrNoRows) {
		return DefaultGraderAgentModelID, nil
	}
	if err != nil {
		return "", err
	}
	return s, nil
}

// UpsertAISettings sets platform AI model preferences; returns the stored values.
func UpsertAISettings(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, imageModelID, courseSetupModelID, notebookFlashcardsModelID, vibeActivityModelID, graderAgentModelID string) (imgOut, courseOut, flashcardsOut, vibeOut, graderOut string, err error) {
	if pool == nil {
		return "", "", "", "", "", errors.New("db pool is nil")
	}
	err = pool.QueryRow(ctx, `
INSERT INTO "user".user_ai_settings (user_id, image_model_id, course_setup_model_id, notebook_flashcards_model_id, vibe_activity_model_id, grader_agent_model_id, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (user_id) DO UPDATE SET
	image_model_id = EXCLUDED.image_model_id,
	course_setup_model_id = EXCLUDED.course_setup_model_id,
	notebook_flashcards_model_id = EXCLUDED.notebook_flashcards_model_id,
	vibe_activity_model_id = EXCLUDED.vibe_activity_model_id,
	grader_agent_model_id = EXCLUDED.grader_agent_model_id,
	updated_at = now()
RETURNING image_model_id, course_setup_model_id, notebook_flashcards_model_id, vibe_activity_model_id, grader_agent_model_id
`, userID, imageModelID, courseSetupModelID, notebookFlashcardsModelID, vibeActivityModelID, graderAgentModelID).Scan(&imgOut, &courseOut, &flashcardsOut, &vibeOut, &graderOut)
	return imgOut, courseOut, flashcardsOut, vibeOut, graderOut, err
}
