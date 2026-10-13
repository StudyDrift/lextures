package homeschoolrecords

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/service/coursefinalgrade"
)

// TranscriptCourse is one enrollment line for a year-end transcript.
type TranscriptCourse struct {
	CourseID   uuid.UUID
	CourseCode string
	Title      string
	Credits    float64
	FinalGrade string
	Percent    *float64
	TermLabel  string
}

// ListTranscriptCourses returns active student enrollments overlapping the school-year window.
func ListTranscriptCourses(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, from, to time.Time) ([]TranscriptCourse, error) {
	rows, err := pool.Query(ctx, `
SELECT
    c.id,
    c.course_code,
    c.title,
    COALESCE(
        (SELECT cs.credits FROM catalog.catalog_sections cs
         WHERE cs.lms_course_id = c.id
         ORDER BY cs.updated_at DESC NULLS LAST LIMIT 1),
        1
    )::float8 AS credits,
    (
        SELECT fgs.final_grade FROM course.final_grade_submissions fgs
        WHERE fgs.enrollment_id = ce.id
        ORDER BY fgs.submitted_at DESC LIMIT 1
    ) AS final_grade,
    COALESCE(tr.name, '') AS term_label,
    c.created_at
FROM course.course_enrollments ce
JOIN course.courses c ON c.id = ce.course_id
LEFT JOIN tenant.terms tr ON tr.id = c.term_id
LEFT JOIN course.enrollment_roles er ON er.role_key = ce.role
WHERE ce.user_id = $1
  AND ce.active = TRUE
  AND (er.is_student_equivalent IS TRUE OR ce.role = 'student')
  AND COALESCE(c.archived, false) = false
  AND (
        (tr.start_date IS NOT NULL AND tr.start_date <= $3::date AND (tr.end_date IS NULL OR tr.end_date >= $2::date))
     OR (tr.start_date IS NULL AND c.created_at < ($3::date + INTERVAL '1 day') AND c.created_at >= $2::date)
  )
ORDER BY tr.start_date NULLS LAST, c.course_code
`, studentID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TranscriptCourse, 0)
	now := time.Now().UTC()
	for rows.Next() {
		var r TranscriptCourse
		var final *string
		var createdAt time.Time
		if err := rows.Scan(&r.CourseID, &r.CourseCode, &r.Title, &r.Credits, &final, &r.TermLabel, &createdAt); err != nil {
			return nil, err
		}
		if final != nil {
			r.FinalGrade = *final
		}
		if pct, err := coursefinalgrade.PercentForStudent(ctx, pool, r.CourseID, studentID, now); err == nil && pct != nil {
			r.Percent = pct
			if r.FinalGrade == "" {
				r.FinalGrade = fmt.Sprintf("%.0f%%", *pct)
			}
		}
		if r.FinalGrade == "" {
			r.FinalGrade = "IP"
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
