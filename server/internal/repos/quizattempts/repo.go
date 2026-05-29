package quizattempts

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuizAttemptRow struct {
	ID                        uuid.UUID
	CourseID                  uuid.UUID
	StructureItemID           uuid.UUID
	StudentUserID             uuid.UUID
	Status                    string
	AttemptNumber             int32
	StartedAt                 time.Time
	SubmittedAt               *time.Time
	CurrentQuestionIndex      int32
	DeadlineAt                *time.Time
	EffectiveTimeLimitSeconds *int32
	ExtendedTimeApplied       bool
}

func GetAttempt(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID) (*QuizAttemptRow, error) {
	var r QuizAttemptRow
	err := pool.QueryRow(ctx, `
SELECT id, course_id, structure_item_id, student_user_id, status, attempt_number, started_at, submitted_at,
       current_question_index, deadline_at, effective_time_limit_seconds, extended_time_applied
FROM course.quiz_attempts
WHERE id = $1
`, attemptID).Scan(
		&r.ID, &r.CourseID, &r.StructureItemID, &r.StudentUserID, &r.Status, &r.AttemptNumber,
		&r.StartedAt, &r.SubmittedAt, &r.CurrentQuestionIndex,
		&r.DeadlineAt, &r.EffectiveTimeLimitSeconds, &r.ExtendedTimeApplied,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func FindInProgressAttempt(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, structureItemID, studentUserID uuid.UUID,
) (*QuizAttemptRow, error) {
	var r QuizAttemptRow
	err := pool.QueryRow(ctx, `
SELECT id, course_id, structure_item_id, student_user_id, status, attempt_number, started_at, submitted_at,
       current_question_index, deadline_at, effective_time_limit_seconds, extended_time_applied
FROM course.quiz_attempts
WHERE course_id = $1 AND structure_item_id = $2 AND student_user_id = $3 AND status = 'in_progress'
ORDER BY started_at DESC
LIMIT 1
`, courseID, structureItemID, studentUserID).Scan(
		&r.ID, &r.CourseID, &r.StructureItemID, &r.StudentUserID, &r.Status, &r.AttemptNumber,
		&r.StartedAt, &r.SubmittedAt, &r.CurrentQuestionIndex,
		&r.DeadlineAt, &r.EffectiveTimeLimitSeconds, &r.ExtendedTimeApplied,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

type InsertAttemptParams struct {
	CourseID                  uuid.UUID
	StructureItemID           uuid.UUID
	StudentUserID             uuid.UUID
	AttemptNumber             int32
	DeadlineAt                *time.Time
	EffectiveTimeLimitSeconds *int32
	ExtendedTimeApplied       bool
}

func InsertAttempt(ctx context.Context, pool *pgxpool.Pool, p InsertAttemptParams) (*QuizAttemptRow, error) {
	var r QuizAttemptRow
	err := pool.QueryRow(ctx, `
INSERT INTO course.quiz_attempts (
  course_id, structure_item_id, student_user_id, attempt_number, status,
  deadline_at, effective_time_limit_seconds, extended_time_applied
) VALUES ($1, $2, $3, $4, 'in_progress', $5, $6, $7)
RETURNING id, course_id, structure_item_id, student_user_id, status, attempt_number, started_at, submitted_at,
          current_question_index, deadline_at, effective_time_limit_seconds, extended_time_applied
`, p.CourseID, p.StructureItemID, p.StudentUserID, p.AttemptNumber,
		p.DeadlineAt, p.EffectiveTimeLimitSeconds, p.ExtendedTimeApplied,
	).Scan(
		&r.ID, &r.CourseID, &r.StructureItemID, &r.StudentUserID, &r.Status, &r.AttemptNumber,
		&r.StartedAt, &r.SubmittedAt, &r.CurrentQuestionIndex,
		&r.DeadlineAt, &r.EffectiveTimeLimitSeconds, &r.ExtendedTimeApplied,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func CountSubmittedAttempts(ctx context.Context, pool *pgxpool.Pool, courseID, structureItemID, studentUserID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::bigint
FROM course.quiz_attempts
WHERE course_id = $1 AND structure_item_id = $2 AND student_user_id = $3 AND status = 'submitted'
`, courseID, structureItemID, studentUserID).Scan(&n)
	return n, err
}

func InsertFocusLossEvent(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID, eventType string, durationMS *int32) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.quiz_focus_loss_events (attempt_id, event_type, duration_ms)
VALUES ($1, $2, $3)
`, attemptID, eventType, durationMS)
	return err
}
