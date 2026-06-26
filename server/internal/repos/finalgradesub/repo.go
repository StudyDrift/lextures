// Package finalgradesub stores and queries final grade submission records (plan 14.5).
package finalgradesub

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Submission is one row in course.final_grade_submissions.
type Submission struct {
	ID               uuid.UUID
	CourseID         uuid.UUID
	EnrollmentID     uuid.UUID
	SubmittedBy      uuid.UUID
	ComputedGrade    string
	FinalGrade       string
	OverrideReason   *string
	SubmissionMethod string
	SubmittedAt      time.Time
	SISAckAt         *time.Time
}

// BulkCreate inserts multiple submission rows within a single transaction.
func BulkCreate(ctx context.Context, pool *pgxpool.Pool, rows []Submission) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, s := range rows {
		_, err := tx.Exec(ctx, `
INSERT INTO course.final_grade_submissions
    (course_id, enrollment_id, submitted_by, computed_grade, final_grade,
     override_reason, submission_method)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`,
			s.CourseID, s.EnrollmentID, s.SubmittedBy, s.ComputedGrade, s.FinalGrade,
			s.OverrideReason, s.SubmissionMethod,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// LatestByCourse returns the most recent submission for each enrollment in a course.
func LatestByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]Submission, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ON (enrollment_id)
    id, course_id, enrollment_id, submitted_by, computed_grade,
    final_grade, override_reason, submission_method, submitted_at, sis_ack_at
FROM course.final_grade_submissions
WHERE course_id = $1
ORDER BY enrollment_id, submitted_at DESC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Submission
	for rows.Next() {
		var s Submission
		if err := rows.Scan(
			&s.ID, &s.CourseID, &s.EnrollmentID, &s.SubmittedBy,
			&s.ComputedGrade, &s.FinalGrade, &s.OverrideReason,
			&s.SubmissionMethod, &s.SubmittedAt, &s.SISAckAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CourseSubmissionStatus summarises whether a course has submitted final grades.
type CourseSubmissionStatus struct {
	CourseID       uuid.UUID
	CourseCode     string
	CourseTitle    string
	InstructorID   *uuid.UUID
	InstructorName string
	SubmittedAt    *time.Time
	TotalStudents  int
	SubmittedCount int
}

// ListStatusByTerm returns submission status for all courses in a term.
func ListStatusByTerm(ctx context.Context, pool *pgxpool.Pool, termID uuid.UUID) ([]CourseSubmissionStatus, error) {
	rows, err := pool.Query(ctx, `
SELECT
    c.id,
    c.course_code,
    c.title,
    instr.user_id,
    COALESCE(u.display_name, u.email, '') AS instructor_name,
    MAX(fgs.submitted_at)                  AS last_submitted_at,
    COUNT(DISTINCT ce.id) FILTER (WHERE ce_er.role_key IS NOT NULL) AS total_students,
    COUNT(DISTINCT fgs.enrollment_id)      AS submitted_count
FROM course.courses c
LEFT JOIN course.course_enrollments instr
    ON instr.course_id = c.id
    AND instr.active = TRUE
LEFT JOIN course.enrollment_roles instr_er
    ON instr_er.role_key = instr.role AND instr_er.is_staff = TRUE
LEFT JOIN "user".users u ON u.id = instr.user_id AND instr_er.role_key IS NOT NULL
LEFT JOIN course.course_enrollments ce
    ON ce.course_id = c.id
    AND ce.active = TRUE
LEFT JOIN course.enrollment_roles ce_er
    ON ce_er.role_key = ce.role AND ce_er.is_student_equivalent = TRUE
LEFT JOIN course.final_grade_submissions fgs ON fgs.course_id = c.id
WHERE c.term_id = $1
GROUP BY c.id, c.course_code, c.title, instr.user_id, u.display_name, u.email
ORDER BY c.course_code
`, termID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CourseSubmissionStatus
	for rows.Next() {
		var s CourseSubmissionStatus
		if err := rows.Scan(
			&s.CourseID, &s.CourseCode, &s.CourseTitle,
			&s.InstructorID, &s.InstructorName,
			&s.SubmittedAt, &s.TotalStudents, &s.SubmittedCount,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
