package course

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/search"
)

// SearchCoursesQuery finds enrolled, non-archived courses matching free text (FTS + ILIKE fallback).
func SearchCoursesQuery(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	q string,
	scopeCourseCode *string,
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

	rows, err := pool.Query(ctx, `
WITH matched AS (
    SELECT
        c.course_code,
        c.title,
        GREATEST(
            ts_rank(c.search_vector, websearch_to_tsquery('english', $2)),
            CASE
                WHEN lower(c.course_code) = lower($2) THEN 1.0
                WHEN c.course_code ILIKE $3 OR c.title ILIKE $3 THEN 0.4
                ELSE 0
            END
        ) AS rank
    FROM course.courses c
    WHERE c.archived = false
      AND c.id IN (
          SELECT e.course_id
          FROM course.course_enrollments e
          WHERE e.user_id = $1 AND e.active
      )
      AND ($4::text IS NULL OR lower(c.course_code) = lower($4))
      AND (
          c.search_vector @@ websearch_to_tsquery('english', $2)
          OR c.course_code ILIKE $3
          OR c.title ILIKE $3
      )
)
SELECT course_code, title, rank, COUNT(*) OVER () AS total
FROM matched
ORDER BY rank DESC, title ASC
LIMIT $5
`, userID, q, like, scopeCourseCode, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []search.QueryResultItem
	var total int
	for rows.Next() {
		var code, title string
		var rank float64
		if err := rows.Scan(&code, &title, &rank, &total); err != nil {
			return nil, 0, err
		}
		out = append(out, search.QueryResultItem{
			ID:       "course:" + code,
			Type:     "course",
			Title:    title,
			Subtitle: code,
			Path:     "/courses/" + url.PathEscape(code),
			Score:    rank,
		})
	}
	return out, total, rows.Err()
}

// SearchContentQuery finds published module items the user can open (FTS + ILIKE fallback).
// editCourseCodes lists course codes where the caller may see unpublished items.
func SearchContentQuery(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	q string,
	scopeCourseCode *string,
	editCourseCodes map[string]struct{},
	allowUnpublishedAll bool,
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
	editCodes := make([]string, 0, len(editCourseCodes))
	for code := range editCourseCodes {
		editCodes = append(editCodes, code)
	}

	rows, err := pool.Query(ctx, `
WITH enrolled AS (
    SELECT e.course_id
    FROM course.course_enrollments e
    WHERE e.user_id = $1 AND e.active
),
matched AS (
    SELECT
        csi.id,
        csi.kind,
        csi.title,
        c.course_code,
        c.title AS course_title,
        GREATEST(
            ts_rank(csi.search_vector, websearch_to_tsquery('english', $2)),
            CASE WHEN csi.title ILIKE $3 THEN 0.35 ELSE 0 END
        ) AS rank
    FROM course.course_structure_items csi
    INNER JOIN course.courses c ON c.id = csi.course_id
    INNER JOIN enrolled e ON e.course_id = c.id
    WHERE c.archived = false
      AND csi.archived = false
      AND csi.kind NOT IN ('module', 'heading')
      AND ($4::text IS NULL OR lower(c.course_code) = lower($4))
      AND (
          csi.search_vector @@ websearch_to_tsquery('english', $2)
          OR csi.title ILIKE $3
      )
      AND (
          csi.published = true
          OR $7::boolean = true
          OR c.course_code = ANY($6::text[])
      )
)
SELECT id, kind, title, course_code, course_title, rank, COUNT(*) OVER () AS total
FROM matched
ORDER BY rank DESC, title ASC
LIMIT $5
`, userID, q, like, scopeCourseCode, limit, editCodes, allowUnpublishedAll)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []search.QueryResultItem
	var total int
	for rows.Next() {
		var id uuid.UUID
		var kind, title, courseCode, courseTitle string
		var rank float64
		if err := rows.Scan(&id, &kind, &title, &courseCode, &courseTitle, &rank, &total); err != nil {
			return nil, 0, err
		}
		subtitle := contentKindLabel(kind)
		if t := strings.TrimSpace(courseTitle); t != "" {
			subtitle = fmt.Sprintf("%s · %s · %s", subtitle, t, courseCode)
		} else {
			subtitle = fmt.Sprintf("%s · %s", subtitle, courseCode)
		}
		out = append(out, search.QueryResultItem{
			ID:       fmt.Sprintf("content:%s:%s", courseCode, id.String()),
			Type:     "content",
			Title:    title,
			Subtitle: subtitle,
			Path:     structureItemPath(courseCode, kind, id),
			Score:    rank,
		})
	}
	return out, total, rows.Err()
}

func contentKindLabel(kind string) string {
	switch kind {
	case "content_page":
		return "Page"
	case "assignment":
		return "Assignment"
	case "quiz":
		return "Quiz"
	case "external_link":
		return "External link"
	case "survey":
		return "Survey"
	case "lti_link":
		return "LTI"
	case "h5p":
		return "H5P"
	case "vibe_activity":
		return "Vibe activity"
	default:
		return "Item"
	}
}

func structureItemPath(courseCode, kind string, itemID uuid.UUID) string {
	cc := url.PathEscape(courseCode)
	id := itemID.String()
	switch kind {
	case "content_page":
		return fmt.Sprintf("/courses/%s/modules/content/%s", cc, id)
	case "assignment":
		return fmt.Sprintf("/courses/%s/modules/assignment/%s", cc, id)
	case "quiz":
		return fmt.Sprintf("/courses/%s/modules/quiz/%s", cc, id)
	case "external_link":
		return fmt.Sprintf("/courses/%s/modules/external-link/%s", cc, id)
	case "survey":
		return fmt.Sprintf("/courses/%s/modules/survey/%s", cc, id)
	case "lti_link":
		return fmt.Sprintf("/courses/%s/modules/lti/%s", cc, id)
	case "h5p":
		return fmt.Sprintf("/courses/%s/modules/h5p/%s", cc, id)
	case "vibe_activity":
		return fmt.Sprintf("/courses/%s/modules/vibe-activity/%s", cc, id)
	default:
		return fmt.Sprintf("/courses/%s/modules", cc)
	}
}
