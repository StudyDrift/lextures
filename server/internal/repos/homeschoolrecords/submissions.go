package homeschoolrecords

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SubmissionSample is a work sample a parent can include in a portfolio.
type SubmissionSample struct {
	SubmissionID      uuid.UUID
	CourseID          uuid.UUID
	CourseCode        string
	CourseTitle       string
	ItemID            uuid.UUID
	ItemTitle         string
	SubmittedAt       time.Time
	BodyPreview       string
	InstructorComment string
	PointsEarned      *float64
}

// ListSubmissionSamples lists the learner's assignment submissions with optional teacher comments.
func ListSubmissionSamples(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, from, to time.Time) ([]SubmissionSample, error) {
	fromT := dateOnlyUTC(from)
	toT := dateOnlyUTC(to).Add(24 * time.Hour)
	rows, err := pool.Query(ctx, `
SELECT
    mas.id,
    mas.course_id,
    c.course_code,
    c.title,
    mas.module_item_id,
    COALESCE(csi.title, 'Assignment') AS item_title,
    mas.submitted_at,
    LEFT(COALESCE(mas.body_text, ''), 500) AS body_preview,
    COALESCE(cg.instructor_comment, COALESCE(mas.revision_feedback, '')) AS teacher_comment,
    cg.points_earned
FROM course.module_assignment_submissions mas
JOIN course.courses c ON c.id = mas.course_id
LEFT JOIN course.course_structure_items csi ON csi.id = mas.module_item_id
LEFT JOIN course.course_grades cg
  ON cg.course_id = mas.course_id
 AND cg.student_user_id = mas.submitted_by
 AND cg.module_item_id = mas.module_item_id
WHERE mas.submitted_by = $1
  AND mas.submitted_at >= $2 AND mas.submitted_at < $3
ORDER BY mas.submitted_at DESC
LIMIT 200
`, studentID, fromT, toT)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SubmissionSample, 0)
	for rows.Next() {
		var s SubmissionSample
		if err := rows.Scan(
			&s.SubmissionID, &s.CourseID, &s.CourseCode, &s.CourseTitle,
			&s.ItemID, &s.ItemTitle, &s.SubmittedAt, &s.BodyPreview,
			&s.InstructorComment, &s.PointsEarned,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSubmissionSamplesByIDs loads specific submissions for the student (portfolio selection).
func GetSubmissionSamplesByIDs(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, ids []uuid.UUID) ([]SubmissionSample, error) {
	if len(ids) == 0 {
		return []SubmissionSample{}, nil
	}
	rows, err := pool.Query(ctx, `
SELECT
    mas.id,
    mas.course_id,
    c.course_code,
    c.title,
    mas.module_item_id,
    COALESCE(csi.title, 'Assignment') AS item_title,
    mas.submitted_at,
    LEFT(COALESCE(mas.body_text, ''), 2000) AS body_preview,
    COALESCE(cg.instructor_comment, COALESCE(mas.revision_feedback, '')) AS teacher_comment,
    cg.points_earned
FROM course.module_assignment_submissions mas
JOIN course.courses c ON c.id = mas.course_id
LEFT JOIN course.course_structure_items csi ON csi.id = mas.module_item_id
LEFT JOIN course.course_grades cg
  ON cg.course_id = mas.course_id
 AND cg.student_user_id = mas.submitted_by
 AND cg.module_item_id = mas.module_item_id
WHERE mas.submitted_by = $1
  AND mas.id = ANY($2::uuid[])
ORDER BY mas.submitted_at DESC
`, studentID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SubmissionSample, 0)
	for rows.Next() {
		var s SubmissionSample
		if err := rows.Scan(
			&s.SubmissionID, &s.CourseID, &s.CourseCode, &s.CourseTitle,
			&s.ItemID, &s.ItemTitle, &s.SubmittedAt, &s.BodyPreview,
			&s.InstructorComment, &s.PointsEarned,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
