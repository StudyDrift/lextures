package studentprogress

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListCompletedAssessmentItemIDs returns published assignment and quiz items the
// learner has finished: an assignment with a submission, or a quiz with a
// submitted attempt.
func ListCompletedAssessmentItemIDs(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT csi.id
FROM course.course_structure_items csi
INNER JOIN course.module_assignment_submissions mas
    ON mas.module_item_id = csi.id
   AND mas.course_id = csi.course_id
   AND mas.submitted_by = $2
WHERE csi.course_id = $1
  AND csi.kind = 'assignment'
  AND csi.published
  AND NOT csi.archived
UNION
SELECT csi.id
FROM course.course_structure_items csi
WHERE csi.course_id = $1
  AND csi.kind = 'quiz'
  AND csi.published
  AND NOT csi.archived
  AND EXISTS (
    SELECT 1
    FROM course.quiz_attempts qa
    WHERE qa.structure_item_id = csi.id
      AND qa.course_id = csi.course_id
      AND qa.student_user_id = $2
      AND qa.status = 'submitted'
  )
`, courseID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
