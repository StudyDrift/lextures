package coursemodulequizzes

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

// PatchWrite holds optional quiz settings updates for PATCH /quizzes/{item_id}.
type PatchWrite struct {
	TimeLimitMinutes *int32
	Questions        *[]coursemodulequiz.QuizQuestion
}

// PatchForCourseItem merges patch fields into the quiz row; returns false when not found.
func PatchForCourseItem(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, w PatchWrite) (bool, error) {
	if w.TimeLimitMinutes == nil && w.Questions == nil {
		return false, errors.New("coursemodulequizzes: patch requires at least one field")
	}
	row, err := GetForCourseItem(ctx, pool, courseID, itemID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}
	timeLimit := row.TimeLimitMinutes
	if w.TimeLimitMinutes != nil {
		timeLimit = w.TimeLimitMinutes
	}
	questions := row.Questions
	if w.Questions != nil {
		questions = *w.Questions
	}
	qJSON, err := json.Marshal(questions)
	if err != nil {
		return false, err
	}
	tag, err := pool.Exec(ctx, `
UPDATE course.module_quizzes q
SET time_limit_minutes = $3,
    questions_json = $4::jsonb,
    updated_at = NOW()
FROM course.course_structure_items c
WHERE q.structure_item_id = c.id
  AND c.course_id = $1
  AND c.id = $2
  AND c.kind = 'quiz'
`, courseID, itemID, timeLimit, qJSON)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
