package provisionalgrades

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProvisionalGradeRow struct {
	ID           uuid.UUID
	SubmissionID uuid.UUID
	GraderID     uuid.UUID
	Score        float64
	RubricData   json.RawMessage
	SubmittedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func scanRow(scanner interface{ Scan(...any) error }) (*ProvisionalGradeRow, error) {
	var r ProvisionalGradeRow
	err := scanner.Scan(
		&r.ID, &r.SubmissionID, &r.GraderID, &r.Score, &r.RubricData, &r.SubmittedAt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListForAssignment returns all provisional grades for submissions on an assignment.
func ListForAssignment(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) ([]ProvisionalGradeRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	rows, err := pool.Query(ctx, `
SELECT pg.id, pg.submission_id, pg.grader_id, pg.score, pg.rubric_data, pg.submitted_at, pg.created_at, pg.updated_at
FROM course.provisional_grades pg
INNER JOIN course.module_assignment_submissions s ON s.id = pg.submission_id
WHERE s.course_id = $1 AND s.module_item_id = $2
ORDER BY pg.submission_id, pg.grader_id
`, courseID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ProvisionalGradeRow, 0)
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// ListForAssignmentByGrader returns provisional grades visible to one grader (own rows only).
func ListForAssignmentByGrader(ctx context.Context, pool *pgxpool.Pool, courseID, itemID, graderID uuid.UUID) ([]ProvisionalGradeRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	rows, err := pool.Query(ctx, `
SELECT pg.id, pg.submission_id, pg.grader_id, pg.score, pg.rubric_data, pg.submitted_at, pg.created_at, pg.updated_at
FROM course.provisional_grades pg
INNER JOIN course.module_assignment_submissions s ON s.id = pg.submission_id
WHERE s.course_id = $1 AND s.module_item_id = $2 AND pg.grader_id = $3
ORDER BY pg.submission_id
`, courseID, itemID, graderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ProvisionalGradeRow, 0)
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// Upsert saves or replaces a grader's provisional score for a submission.
func Upsert(ctx context.Context, pool *pgxpool.Pool, submissionID, graderID uuid.UUID, score float64) error {
	if pool == nil {
		return errors.New("nil pool")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.provisional_grades (submission_id, grader_id, score, submitted_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (submission_id, grader_id) DO UPDATE SET
	score = EXCLUDED.score,
	submitted_at = NOW(),
	updated_at = NOW()
`, submissionID, graderID, score)
	return err
}

// GetForSubmissionGrader returns one provisional grade row when present.
func GetForSubmissionGrader(ctx context.Context, pool *pgxpool.Pool, submissionID, graderID uuid.UUID) (*ProvisionalGradeRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	r, err := scanRow(pool.QueryRow(ctx, `
SELECT id, submission_id, grader_id, score, rubric_data, submitted_at, created_at, updated_at
FROM course.provisional_grades
WHERE submission_id = $1 AND grader_id = $2
`, submissionID, graderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}