package quizattempts

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ResponseRow is one row to insert into course.quiz_responses.
type ResponseRow struct {
	QuestionIndex  int32
	QuestionID     string
	QuestionType   string
	PromptSnapshot string
	ResponseJSON   json.RawMessage
	IsCorrect      *bool
	PointsAwarded  float64
	MaxPoints      float64
	Locked         bool
}

// AttemptResultRow extends attempt metadata with score fields for results/submit.
type AttemptResultRow struct {
	QuizAttemptRow
	IsAdaptive            bool
	PointsEarned          *float64
	PointsPossible        *float64
	ScorePercent          *float32
	AcademicIntegrityFlag bool
	AdaptiveHistoryJSON   json.RawMessage
}

func ReplaceResponses(ctx context.Context, tx pgx.Tx, attemptID uuid.UUID, rows []ResponseRow) error {
	if _, err := tx.Exec(ctx, `DELETE FROM course.quiz_responses WHERE attempt_id = $1`, attemptID); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.quiz_responses (
  attempt_id, question_index, question_id, question_type, prompt_snapshot,
  response_json, is_correct, points_awarded, max_points, locked
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`, attemptID, r.QuestionIndex, nullString(r.QuestionID), r.QuestionType, nullString(r.PromptSnapshot),
			r.ResponseJSON, r.IsCorrect, r.PointsAwarded, r.MaxPoints, r.Locked); err != nil {
			return err
		}
	}
	return nil
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type FinalizeSubmitParams struct {
	AttemptID             uuid.UUID
	SubmittedAt           time.Time
	PointsEarned          float64
	PointsPossible        float64
	ScorePercent          float32
	AcademicIntegrityFlag bool
	IsAdaptive            bool
	AdaptiveHistoryJSON   json.RawMessage
	CurrentQuestionIndex  *int32
}

// FinalizeAttemptSubmitted marks an in-progress attempt as submitted with score.
func FinalizeAttemptSubmitted(ctx context.Context, tx pgx.Tx, p FinalizeSubmitParams) (bool, error) {
	var idx any
	if p.CurrentQuestionIndex != nil {
		idx = *p.CurrentQuestionIndex
	}
	tag, err := tx.Exec(ctx, `
UPDATE course.quiz_attempts
SET status = 'submitted',
    submitted_at = $2,
    points_earned = $3,
    points_possible = $4,
    score_percent = $5,
    academic_integrity_flag = $6,
    is_adaptive = $7,
    adaptive_history_json = $8,
    current_question_index = COALESCE($9, current_question_index)
WHERE id = $1 AND status = 'in_progress'
`, p.AttemptID, p.SubmittedAt, p.PointsEarned, p.PointsPossible, p.ScorePercent,
		p.AcademicIntegrityFlag, p.IsAdaptive, nullJSON(p.AdaptiveHistoryJSON), idx)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func nullJSON(b json.RawMessage) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// UpsertLockedResponse saves one lockdown-mode answer and advances the attempt index.
func UpsertLockedResponse(ctx context.Context, tx pgx.Tx, attemptID uuid.UUID, questionIndex int32, row ResponseRow) error {
	row.Locked = true
	_, err := tx.Exec(ctx, `
INSERT INTO course.quiz_responses (
  attempt_id, question_index, question_id, question_type, prompt_snapshot,
  response_json, is_correct, points_awarded, max_points, locked
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (attempt_id, question_index) DO UPDATE SET
  question_id = EXCLUDED.question_id,
  question_type = EXCLUDED.question_type,
  prompt_snapshot = EXCLUDED.prompt_snapshot,
  response_json = EXCLUDED.response_json,
  is_correct = EXCLUDED.is_correct,
  points_awarded = EXCLUDED.points_awarded,
  max_points = EXCLUDED.max_points,
  locked = EXCLUDED.locked
`, attemptID, questionIndex, nullString(row.QuestionID), row.QuestionType, nullString(row.PromptSnapshot),
		row.ResponseJSON, row.IsCorrect, row.PointsAwarded, row.MaxPoints, row.Locked)
	return err
}

func SetAttemptQuestionIndex(ctx context.Context, tx pgx.Tx, attemptID uuid.UUID, idx int32) error {
	_, err := tx.Exec(ctx, `
UPDATE course.quiz_attempts SET current_question_index = $2 WHERE id = $1 AND status = 'in_progress'
`, attemptID, idx)
	return err
}

func GetAttemptResult(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID) (*AttemptResultRow, error) {
	var r AttemptResultRow
	err := pool.QueryRow(ctx, `
SELECT id, course_id, structure_item_id, student_user_id, status, attempt_number, started_at, submitted_at,
       current_question_index, deadline_at, effective_time_limit_seconds, extended_time_applied,
       is_adaptive, points_earned, points_possible, score_percent, academic_integrity_flag, adaptive_history_json
FROM course.quiz_attempts
WHERE id = $1
`, attemptID).Scan(
		&r.ID, &r.CourseID, &r.StructureItemID, &r.StudentUserID, &r.Status, &r.AttemptNumber,
		&r.StartedAt, &r.SubmittedAt, &r.CurrentQuestionIndex,
		&r.DeadlineAt, &r.EffectiveTimeLimitSeconds, &r.ExtendedTimeApplied,
		&r.IsAdaptive, &r.PointsEarned, &r.PointsPossible, &r.ScorePercent,
		&r.AcademicIntegrityFlag, &r.AdaptiveHistoryJSON,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func ListResponses(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID) ([]ResponseRow, error) {
	rows, err := pool.Query(ctx, `
SELECT question_index, COALESCE(question_id, ''), question_type, COALESCE(prompt_snapshot, ''),
       response_json, is_correct, COALESCE(points_awarded, 0), max_points, locked
FROM course.quiz_responses
WHERE attempt_id = $1
ORDER BY question_index ASC
`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ResponseRow
	for rows.Next() {
		var r ResponseRow
		if err := rows.Scan(
			&r.QuestionIndex, &r.QuestionID, &r.QuestionType, &r.PromptSnapshot,
			&r.ResponseJSON, &r.IsCorrect, &r.PointsAwarded, &r.MaxPoints, &r.Locked,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
