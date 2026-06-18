package course

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ArchivedCourseRow is one archived course for the global settings archive table.
type ArchivedCourseRow struct {
	ID               uuid.UUID  `json:"id"`
	CourseCode       string     `json:"courseCode"`
	Title            string     `json:"title"`
	ArchivedAt       time.Time  `json:"archivedAt"`
	ArchivedByUserID *uuid.UUID `json:"archivedByUserId,omitempty"`
	ArchivedByName   *string    `json:"archivedByName,omitempty"`
	ArchivedByEmail  *string    `json:"archivedByEmail,omitempty"`
}

// ListArchivedInOrg returns archived courses in orgID, newest first.
func ListArchivedInOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]ArchivedCourseRow, error) {
	const q = `
SELECT
    c.id,
    c.course_code,
    c.title,
    c.archived_at,
    c.archived_by_user_id,
    NULLIF(TRIM(COALESCE(u.display_name,
        TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')))), '') AS archived_by_name,
    u.email AS archived_by_email
FROM course.courses c
LEFT JOIN "user".users u ON u.id = c.archived_by_user_id
WHERE c.org_id = $1
  AND c.archived = TRUE
ORDER BY c.archived_at DESC NULLS LAST, c.updated_at DESC
`
	rows, err := pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ArchivedCourseRow, 0)
	for rows.Next() {
		var row ArchivedCourseRow
		var archivedAt *time.Time
		if err := rows.Scan(
			&row.ID,
			&row.CourseCode,
			&row.Title,
			&archivedAt,
			&row.ArchivedByUserID,
			&row.ArchivedByName,
			&row.ArchivedByEmail,
		); err != nil {
			return nil, err
		}
		if archivedAt != nil {
			row.ArchivedAt = *archivedAt
		}
		out = append(out, row)
	}
	return out, rows.Err()
}