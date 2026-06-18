package quizattempts

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttemptListRow struct {
	ID                 uuid.UUID
	StudentUserID      uuid.UUID
	StudentDisplayName string
	AttemptNumber      int32
	SubmittedAt        time.Time
	ScorePercent       *float32
	PointsEarned       float64
	PointsPossible     float64
	NeedsManualGrading bool
}

func ListSubmittedAttemptsForItem(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, structureItemID uuid.UUID,
	studentUserID *uuid.UUID,
) ([]AttemptListRow, error) {
	var studentClause string
	args := []any{courseID, structureItemID}
	if studentUserID != nil {
		studentClause = "AND qa.student_user_id = $3"
		args = append(args, *studentUserID)
	}
	rows, err := pool.Query(ctx, `
SELECT qa.id, qa.student_user_id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email, 'Student'),
       qa.attempt_number, qa.submitted_at, qa.score_percent,
       COALESCE(qa.points_earned, 0), COALESCE(qa.points_possible, 0),
       EXISTS (
         SELECT 1
         FROM course.quiz_responses qr
         WHERE qr.attempt_id = qa.id
           AND qr.question_type IN ('essay', 'file_upload', 'audio_response', 'video_response', 'code', 'hotspot', 'formula')
           AND qr.response_json::text NOT IN ('{}', 'null')
           AND qr.is_correct IS NULL
           AND COALESCE(qr.points_awarded, 0) < qr.max_points - 0.0001
       )
FROM course.quiz_attempts qa
INNER JOIN "user".users u ON u.id = qa.student_user_id
WHERE qa.course_id = $1 AND qa.structure_item_id = $2 AND qa.status = 'submitted' `+studentClause+`
ORDER BY qa.submitted_at DESC NULLS LAST, qa.attempt_number DESC
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AttemptListRow, 0)
	for rows.Next() {
		var row AttemptListRow
		if err := rows.Scan(
			&row.ID, &row.StudentUserID, &row.StudentDisplayName, &row.AttemptNumber,
			&row.SubmittedAt, &row.ScorePercent, &row.PointsEarned, &row.PointsPossible, &row.NeedsManualGrading,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}