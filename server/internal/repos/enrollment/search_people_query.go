package enrollment

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/search"
)

// SearchPeopleQuery lists roster people matching free text in courses the requester is enrolled in.
func SearchPeopleQuery(
	ctx context.Context,
	pool *pgxpool.Pool,
	requesterUserID uuid.UUID,
	q string,
	scopeCourseCode *string,
	rosterCourseCodes map[string]struct{},
	gradebookCourseCodes map[string]struct{},
	limit int,
) ([]search.QueryResultItem, int, error) {
	q = strings.TrimSpace(q)
	if q == "" || limit <= 0 {
		return nil, 0, nil
	}
	if limit > 50 {
		limit = 50
	}
	like := "%" + q + "%"
	rosterCodes := make([]string, 0, len(rosterCourseCodes))
	for code := range rosterCourseCodes {
		rosterCodes = append(rosterCodes, code)
	}

	rows, err := pool.Query(ctx, `
WITH matched AS (
    SELECT
        u.id AS user_id,
        u.email,
        u.display_name,
        COALESCE(er.display_name, initcap(ce.role)) AS role_label,
        c.course_code,
        c.title AS course_title,
        GREATEST(
            ts_rank(
                to_tsvector(
                    'english',
                    coalesce(u.display_name, '') || ' ' || coalesce(u.email, '')
                ),
                websearch_to_tsquery('english', $2)
            ),
            CASE
                WHEN u.email ILIKE $3 OR u.display_name ILIKE $3 THEN 0.4
                ELSE 0
            END
        ) AS rank
    FROM course.course_enrollments ce
    INNER JOIN course.courses c ON c.id = ce.course_id
    INNER JOIN "user".users u ON u.id = ce.user_id
    LEFT JOIN course.enrollment_roles er ON er.role_key = ce.role
    WHERE c.archived = false
      AND ce.active
      AND c.id IN (
          SELECT ce2.course_id
          FROM course.course_enrollments ce2
          WHERE ce2.user_id = $1 AND ce2.active
      )
      AND ($4::text IS NULL OR lower(c.course_code) = lower($4))
      AND (
          cardinality($6::text[]) = 0
          OR c.course_code = ANY($6::text[])
      )
      AND (
          to_tsvector('english', coalesce(u.display_name, '') || ' ' || coalesce(u.email, ''))
              @@ websearch_to_tsquery('english', $2)
          OR u.email ILIKE $3
          OR u.display_name ILIKE $3
      )
)
SELECT user_id, email, display_name, role_label, course_code, course_title, rank, COUNT(*) OVER () AS total
FROM matched
ORDER BY rank DESC, course_title ASC, email ASC
LIMIT $5
`, requesterUserID, q, like, scopeCourseCode, limit, rosterCodes)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []search.QueryResultItem
	var total int
	for rows.Next() {
		var userID uuid.UUID
		var email, roleLabel, courseCode, courseTitle string
		var display sql.NullString
		var rank float64
		if err := rows.Scan(&userID, &email, &display, &roleLabel, &courseCode, &courseTitle, &rank, &total); err != nil {
			return nil, 0, err
		}
		title := email
		if display.Valid && strings.TrimSpace(display.String) != "" {
			title = display.String
		}
		subtitle := fmt.Sprintf("%s · %s · %s", courseTitle, courseCode, roleLabel)
		path := fmt.Sprintf("/courses/%s/enrollments", url.PathEscape(courseCode))
		if _, ok := gradebookCourseCodes[courseCode]; ok {
			path = fmt.Sprintf(
				"/courses/%s/gradebook?student=%s",
				url.PathEscape(courseCode),
				url.QueryEscape(userID.String()),
			)
		}
		out = append(out, search.QueryResultItem{
			ID:       fmt.Sprintf("person:%s:%s:%s", userID.String(), courseCode, roleLabel),
			Type:     "person",
			Title:    title,
			Subtitle: subtitle,
			Path:     path,
			Score:    rank,
		})
	}
	return out, total, rows.Err()
}
