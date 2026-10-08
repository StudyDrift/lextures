package coursestandards

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/sbg"
)

// ErrStudentNotEnrolled means the user is not an active student-equivalent
// enrollment on the course. Callers map this to 404.
var ErrStudentNotEnrolled = errors.New("coursestandards: student not enrolled")

// MasteryTranscript is one student's standards proficiency for a course PDF.
type MasteryTranscript struct {
	CourseTitle   string
	CourseCode    string
	StudentUserID uuid.UUID
	StudentName   string
	Rows          []sbg.SbgMasteryTranscriptRow
}

// LoadMasteryTranscript loads the course title, the enrolled student's display
// name, and every course standard with that student's cached proficiency.
// Rows is never nil.
func LoadMasteryTranscript(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, courseCode string, studentID uuid.UUID) (MasteryTranscript, error) {
	out := MasteryTranscript{
		CourseCode:    courseCode,
		StudentUserID: studentID,
		Rows:          []sbg.SbgMasteryTranscriptRow{},
	}
	err := pool.QueryRow(ctx, `
		SELECT c.title,
		       CASE WHEN ce.role = 'test_student' THEN 'Test Student'
		            ELSE COALESCE(NULLIF(TRIM(u.display_name), ''), u.email)
		       END
		FROM course.courses c
		INNER JOIN course.course_enrollments ce
		        ON ce.course_id = c.id AND ce.user_id = $2 AND ce.active
		INNER JOIN "user".users u ON u.id = ce.user_id
		INNER JOIN course.enrollment_roles er
		        ON er.role_key = ce.role AND er.is_student_equivalent = true
		WHERE c.id = $1
		LIMIT 1
	`, courseID, studentID).Scan(&out.CourseTitle, &out.StudentName)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrStudentNotEnrolled
	}
	if err != nil {
		return out, err
	}

	rows, err := pool.Query(ctx, `
		SELECT s.id, s.external_id, s.description,
		       p.proficiency, COALESCE(p.level_label, ''), p.last_assessed
		FROM course.course_standards s
		LEFT JOIN course.student_standard_proficiencies p
		       ON p.standard_id = s.id
		      AND p.course_id = s.course_id
		      AND p.student_id = $2
		WHERE s.course_id = $1
		ORDER BY s."position" ASC, s.id ASC
	`, courseID, studentID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var row sbg.SbgMasteryTranscriptRow
		if err := rows.Scan(&row.StandardID, &row.ExternalID, &row.Description, &row.Proficiency, &row.LevelLabel, &row.LastAssessed); err != nil {
			return out, err
		}
		out.Rows = append(out.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	return out, nil
}
