package attendance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CourseHasAttendanceInRange reports whether the course has any attendance records in [start, end].
func CourseHasAttendanceInRange(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, start, end time.Time) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT
    EXISTS (
        SELECT 1
        FROM course.attendance_records ar
        JOIN course.course_sections cs ON cs.id = ar.section_id AND cs.course_id = $1
        WHERE ar.date >= $2::date AND ar.date <= $3::date
    )
    OR EXISTS (
        SELECT 1
        FROM course.attendance_sessions s
        JOIN course.attendance_session_records r ON r.session_id = s.id
        WHERE s.course_id = $1
          AND s.session_date >= $2::date AND s.session_date <= $3::date
          AND r.status <> 'not_recorded'
    )
`, courseID, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&exists)
	return exists, err
}

// AbsenceCountsForCourseStudents returns per-student absence totals for a course in [start, end].
// Students with zero absences are omitted from the map; callers default to 0 when attendance is known.
func AbsenceCountsForCourseStudents(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, start, end time.Time) (map[uuid.UUID]int, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	rows, err := pool.Query(ctx, `
SELECT student_id, SUM(cnt)::int AS absences
FROM (
    SELECT ar.student_id, COUNT(*)::bigint AS cnt
    FROM course.attendance_records ar
    JOIN course.attendance_codes ac ON ac.id = ar.code_id
    JOIN course.course_sections cs ON cs.id = ar.section_id AND cs.course_id = $1
    WHERE ac.category = 'absent'
      AND ar.date >= $2::date AND ar.date <= $3::date
    GROUP BY ar.student_id

    UNION ALL

    SELECT r.student_user_id AS student_id, COUNT(*)::bigint AS cnt
    FROM course.attendance_session_records r
    JOIN course.attendance_sessions s ON s.id = r.session_id
    WHERE s.course_id = $1
      AND r.status = 'absent'
      AND s.session_date >= $2::date AND s.session_date <= $3::date
    GROUP BY r.student_user_id
) combined
GROUP BY student_id
`, courseID, startStr, endStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uuid.UUID]int)
	for rows.Next() {
		var studentID uuid.UUID
		var count int
		if err := rows.Scan(&studentID, &count); err != nil {
			return nil, err
		}
		out[studentID] = count
	}
	return out, rows.Err()
}