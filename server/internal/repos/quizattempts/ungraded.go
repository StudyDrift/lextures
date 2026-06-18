package quizattempts

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UngradedQuizCount struct {
	StructureItemID uuid.UUID
	Title           string
	UngradedCount   int64
}

// ListUngradedCountsForCourse returns quizzes with submitted attempts that still need manual question grading.
func ListUngradedCountsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]UngradedQuizCount, error) {
	rows, err := pool.Query(ctx, `
SELECT si.id, si.title, COUNT(qa.id)::bigint
FROM course.course_structure_items si
INNER JOIN course.quiz_attempts qa
	ON qa.structure_item_id = si.id AND qa.course_id = $1 AND qa.status = 'submitted'
WHERE si.course_id = $1
  AND si.kind = 'quiz'
  AND EXISTS (
    SELECT 1
    FROM course.quiz_responses qr
    WHERE qr.attempt_id = qa.id
      AND qr.question_type IN ('essay', 'file_upload', 'audio_response', 'video_response', 'code', 'hotspot', 'formula')
      AND qr.response_json::text NOT IN ('{}', 'null')
      AND qr.is_correct IS NULL
      AND COALESCE(qr.points_awarded, 0) < qr.max_points - 0.0001
  )
GROUP BY si.id, si.title
HAVING COUNT(qa.id) > 0
ORDER BY MIN(qa.submitted_at) ASC NULLS LAST, si.title ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]UngradedQuizCount, 0)
	for rows.Next() {
		var row UngradedQuizCount
		if err := rows.Scan(&row.StructureItemID, &row.Title, &row.UngradedCount); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}