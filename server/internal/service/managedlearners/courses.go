package managedlearners

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DependentCourseEnrollment is one active student enrollment for a managed learner.
type DependentCourseEnrollment struct {
	DependentID  string `json:"dependentId"`
	DisplayName  string `json:"displayName"`
	CourseCode   string `json:"courseCode"`
	CourseTitle  string `json:"courseTitle"`
	EnrollmentID string `json:"enrollmentId"`
}

// ListCourseEnrollments returns active student-equivalent enrollments for the actor's managed learners.
// Used to deep-link from the Learners page to each child's course progress.
func ListCourseEnrollments(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID) ([]DependentCourseEnrollment, error) {
	_, orgID, err := assertCanManageDependents(ctx, pool, actorID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT u.id::text, COALESCE(u.display_name, ''), c.course_code, c.title, ce.id::text
FROM "user".parent_student_links l
INNER JOIN "user".users u ON u.id = l.student_user_id
INNER JOIN course.course_enrollments ce ON ce.user_id = u.id AND ce.active
INNER JOIN course.courses c ON c.id = ce.course_id AND c.archived = false
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE l.parent_user_id = $1 AND l.org_id = $2
  AND l.status IN ('active', 'pending')
  AND u.account_type = 'managed'
  AND u.deactivated_at IS NULL
ORDER BY u.display_name NULLS LAST, c.title ASC
`, actorID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DependentCourseEnrollment, 0)
	for rows.Next() {
		var row DependentCourseEnrollment
		if err := rows.Scan(&row.DependentID, &row.DisplayName, &row.CourseCode, &row.CourseTitle, &row.EnrollmentID); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
