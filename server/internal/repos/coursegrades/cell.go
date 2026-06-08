package coursegrades

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CellRow is one gradebook cell including rubric scores and instructor feedback.
type CellRow struct {
	PointsEarned      *float64
	RubricScoresJSON  []byte
	InstructorComment *string
	PostedAt          *time.Time
	Excused           bool
}

func GetCell(ctx context.Context, pool *pgxpool.Pool, courseID, studentID, itemID uuid.UUID) (*CellRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var pts *float64
	var rubricJSON []byte
	var comment *string
	var posted *time.Time
	var excused bool
	err := pool.QueryRow(ctx, `
SELECT points_earned, rubric_scores_json, instructor_comment, posted_at, excused
FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, courseID, studentID, itemID).Scan(&pts, &rubricJSON, &comment, &posted, &excused)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &CellRow{
		PointsEarned:      pts,
		RubricScoresJSON:  rubricJSON,
		InstructorComment: comment,
		PostedAt:          posted,
		Excused:           excused,
	}, nil
}

// UpsertCell saves points, optional rubric JSON, and instructor comment for one cell.
func UpsertCell(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, studentID, itemID uuid.UUID,
	points float64,
	rubricJSON []byte,
	comment *string,
	postingPolicy string,
) error {
	if pool == nil {
		return errors.New("nil pool")
	}
	if postingPolicy == "automatic" {
		_, err := pool.Exec(ctx, `
INSERT INTO course.course_grades (
	course_id, student_user_id, module_item_id, points_earned, rubric_scores_json, instructor_comment, updated_at, posted_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
ON CONFLICT (student_user_id, module_item_id) DO UPDATE SET
	course_id = EXCLUDED.course_id,
	points_earned = EXCLUDED.points_earned,
	rubric_scores_json = EXCLUDED.rubric_scores_json,
	instructor_comment = EXCLUDED.instructor_comment,
	updated_at = NOW(),
	posted_at = COALESCE(course.course_grades.posted_at, NOW())
`, courseID, studentID, itemID, points, nullableJSON(rubricJSON), comment)
		return err
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.course_grades (
	course_id, student_user_id, module_item_id, points_earned, rubric_scores_json, instructor_comment, updated_at, posted_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NULL)
ON CONFLICT (student_user_id, module_item_id) DO UPDATE SET
	course_id = EXCLUDED.course_id,
	points_earned = EXCLUDED.points_earned,
	rubric_scores_json = EXCLUDED.rubric_scores_json,
	instructor_comment = EXCLUDED.instructor_comment,
	updated_at = NOW(),
	posted_at = course.course_grades.posted_at
`, courseID, studentID, itemID, points, nullableJSON(rubricJSON), comment)
	return err
}

func DeleteCell(ctx context.Context, pool *pgxpool.Pool, courseID, studentID, itemID uuid.UUID) error {
	if pool == nil {
		return errors.New("nil pool")
	}
	_, err := pool.Exec(ctx, `
DELETE FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, courseID, studentID, itemID)
	return err
}

func ParseRubricScoresMap(raw []byte) (map[string]float64, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var m map[string]float64
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
