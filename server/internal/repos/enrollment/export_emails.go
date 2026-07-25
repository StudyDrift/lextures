package enrollment

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ExportEmailRole is one roster row for course JSON export (email + enrollment role + optional display name).
type ExportEmailRole struct {
	Email       string
	Role        string
	DisplayName *string
}

// ListEmailRolesForCourseExport returns enrollments for a course code for the course JSON export bundle.
// Matches Rust `list_email_roles_for_course_export`.
func ListEmailRolesForCourseExport(ctx context.Context, pool *pgxpool.Pool, courseCode string) ([]ExportEmailRole, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	rows, err := pool.Query(ctx, `
SELECT
	u.email,
	ce.role,
	NULLIF(TRIM(u.display_name), '') AS display_name
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE c.course_code = $1
ORDER BY lower(u.email) ASC, ce.role ASC
`, courseCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ExportEmailRole
	for rows.Next() {
		var email, role string
		var display *string
		if err := rows.Scan(&email, &role, &display); err != nil {
			return nil, err
		}
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		out = append(out, ExportEmailRole{
			Email:       email,
			Role:        role,
			DisplayName: display,
		})
	}
	return out, rows.Err()
}
