package quizattempts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/service/quizattemptgrading"
)

const manualGradingResponseClause = `
  qr.question_type IN (%s)
  AND (
    qr.response_json::text NOT IN ('{}', 'null')
    OR NULLIF(TRIM(qr.prompt_snapshot), '') IS NOT NULL
  )
  AND qr.is_correct IS NULL
  AND COALESCE(qr.points_awarded, 0) < qr.max_points - 0.0001
`

func manualGradingResponseSQL() string {
	return ManualGradingResponseSQL()
}

// ManualGradingResponseSQL returns the shared WHERE fragment for unscored manual quiz responses.
func ManualGradingResponseSQL() string {
	return fmt.Sprintf(manualGradingResponseClause, quizattemptgrading.ManualGradingQuestionTypesSQL())
}

// AttemptNeedsManualGrading reports whether a submitted attempt still has unscored manual questions.
func AttemptNeedsManualGrading(ctx context.Context, q querier, attemptID uuid.UUID) (bool, error) {
	var needs bool
	err := q.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM course.quiz_responses qr
  WHERE qr.attempt_id = $1
    AND `+manualGradingResponseSQL()+`
)`, attemptID).Scan(&needs)
	return needs, err
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// UpdateResponseManualGrade sets points and correctness for one response row.
func UpdateResponseManualGrade(
	ctx context.Context,
	tx pgx.Tx,
	attemptID uuid.UUID,
	questionIndex int32,
	pointsAwarded, maxPoints float64,
) error {
	isCorrect := quizattemptgrading.CorrectnessFromManualPoints(pointsAwarded, maxPoints)
	tag, err := tx.Exec(ctx, `
UPDATE course.quiz_responses
SET points_awarded = $3,
    is_correct = $4
WHERE attempt_id = $1 AND question_index = $2
`, attemptID, questionIndex, pointsAwarded, isCorrect)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UpsertResponseManualGrade sets or creates a response row so instructors can override any quiz question score.
func UpsertResponseManualGrade(
	ctx context.Context,
	tx pgx.Tx,
	attemptID uuid.UUID,
	row ResponseRow,
	pointsAwarded float64,
) error {
	isCorrect := quizattemptgrading.CorrectnessFromManualPoints(pointsAwarded, row.MaxPoints)
	_, err := tx.Exec(ctx, `
INSERT INTO course.quiz_responses (
  attempt_id, question_index, question_id, question_type, prompt_snapshot,
  response_json, is_correct, points_awarded, max_points, locked
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false)
ON CONFLICT (attempt_id, question_index) DO UPDATE SET
  points_awarded = EXCLUDED.points_awarded,
  is_correct = EXCLUDED.is_correct
`, attemptID, row.QuestionIndex, nullString(row.QuestionID), row.QuestionType, nullString(row.PromptSnapshot),
		row.ResponseJSON, isCorrect, pointsAwarded, row.MaxPoints)
	return err
}

// UpdateAttemptScoreTotals recomputes attempt earned/possible/percent from stored responses.
func UpdateAttemptScoreTotals(ctx context.Context, tx pgx.Tx, attemptID uuid.UUID) (earned, possible float64, score float32, err error) {
	earned, possible, err = SumResponsePointsForAttempt(ctx, tx, attemptID)
	if err != nil {
		return 0, 0, 0, err
	}
	score = quizattemptgrading.ScorePercent(earned, possible)
	_, err = tx.Exec(ctx, `
UPDATE course.quiz_attempts
SET points_earned = $2,
    points_possible = $3,
    score_percent = $4
WHERE id = $1 AND status = 'submitted'
`, attemptID, earned, possible, score)
	return earned, possible, score, err
}

// PolicyPointsForStudent picks the gradebook score from submitted attempts using the quiz policy.
func PolicyPointsForStudent(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, structureItemID, studentID uuid.UUID,
	policy string,
) (points float64, ready bool, err error) {
	rows, err := pool.Query(ctx, `
SELECT qa.id, COALESCE(qa.points_earned, 0), COALESCE(qa.score_percent, 0), qa.attempt_number, qa.submitted_at
FROM course.quiz_attempts qa
WHERE qa.course_id = $1 AND qa.structure_item_id = $2 AND qa.student_user_id = $3 AND qa.status = 'submitted'
ORDER BY qa.submitted_at ASC NULLS LAST, qa.attempt_number ASC
`, courseID, structureItemID, studentID)
	if err != nil {
		return 0, false, err
	}
	defer rows.Close()

	type attemptScore struct {
		id     uuid.UUID
		points float64
		number int32
	}
	var attempts []attemptScore
	for rows.Next() {
		var id uuid.UUID
		var pts float64
		var pct float32
		var num int32
		var submittedAt any
		if err := rows.Scan(&id, &pts, &pct, &num, &submittedAt); err != nil {
			return 0, false, err
		}
		needs, err := AttemptNeedsManualGrading(ctx, pool, id)
		if err != nil {
			return 0, false, err
		}
		if needs {
			continue
		}
		attempts = append(attempts, attemptScore{id: id, points: pts, number: num})
	}
	if err := rows.Err(); err != nil {
		return 0, false, err
	}
	if len(attempts) == 0 {
		return 0, false, nil
	}
	switch policy {
	case "highest":
		best := attempts[0].points
		for _, a := range attempts[1:] {
			if a.points > best {
				best = a.points
			}
		}
		return best, true, nil
	case "first":
		return attempts[0].points, true, nil
	case "average":
		var sum float64
		for _, a := range attempts {
			sum += a.points
		}
		return sum / float64(len(attempts)), true, nil
	default: // latest
		return attempts[len(attempts)-1].points, true, nil
	}
}