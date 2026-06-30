// Package adminsearchrepo provides org-scoped Postgres full-text search queries (plan 18.4).
package adminsearchrepo

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/adminsearch"
)

const systemUserID = "a0000000-0000-4000-8000-000000000001"

// SearchUsers finds users in orgID matching free text (FTS + pg_trgm fuzzy).
func SearchUsers(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, q string, limit, offset int) ([]adminsearch.Result, int64, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	like := "%" + q + "%"

	rows, err := pool.Query(ctx, `
WITH matched AS (
    SELECT
        u.id,
        COALESCE(
            NULLIF(TRIM(COALESCE(u.display_name, '')), ''),
            TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, ''))
        ) AS name,
        u.email,
        GREATEST(
            ts_rank(u.search_vector, websearch_to_tsquery('english', $2)),
            CASE WHEN similarity(u.email, $2) >= 0.3 THEN similarity(u.email, $2) ELSE 0 END,
            CASE WHEN similarity(COALESCE(u.display_name, ''), $2) >= 0.3 THEN similarity(COALESCE(u.display_name, ''), $2) ELSE 0 END,
            CASE WHEN similarity(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), $2) >= 0.3
                THEN similarity(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), $2)
                ELSE 0 END,
            CASE WHEN u.email ILIKE $3 OR COALESCE(u.display_name, '') ILIKE $3 THEN 0.35 ELSE 0 END
        ) AS rank
    FROM "user".users u
    WHERE u.org_id = $1
      AND u.id <> $4::uuid
      AND (
          u.search_vector @@ websearch_to_tsquery('english', $2)
          OR u.email ILIKE $3
          OR similarity(u.email, $2) >= 0.3
          OR similarity(COALESCE(u.display_name, ''), $2) >= 0.3
          OR similarity(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), $2) >= 0.3
      )
)
SELECT
    id,
    name,
    email,
    rank,
    ts_headline(
        'english',
        COALESCE(NULLIF(TRIM(name), ''), email),
        websearch_to_tsquery('english', $2),
        'MaxWords=12, MinWords=2, ShortWord=2'
    ) AS snippet,
    COUNT(*) OVER () AS total
FROM matched
ORDER BY rank DESC, name ASC, email ASC
LIMIT $5 OFFSET $6
`, orgID, q, like, systemUserID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []adminsearch.Result
	var total int64
	for rows.Next() {
		var id uuid.UUID
		var name, email, snippet string
		var rank float64
		if err := rows.Scan(&id, &name, &email, &rank, &snippet, &total); err != nil {
			return nil, 0, err
		}
		title := strings.TrimSpace(name)
		if title == "" {
			title = email
		}
		out = append(out, adminsearch.Result{
			ID:       id.String(),
			Type:     "users",
			Title:    title,
			Subtitle: email,
			Snippet:  snippet,
			Path:     "/org-admin/users?q=" + url.QueryEscape(email),
			Score:    rank,
		})
	}
	return out, total, rows.Err()
}

// SearchCourses finds courses in orgID matching free text (FTS + pg_trgm fuzzy).
func SearchCourses(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, q string, limit, offset int) ([]adminsearch.Result, int64, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	like := "%" + q + "%"

	rows, err := pool.Query(ctx, `
WITH matched AS (
    SELECT
        c.id,
        c.course_code,
        c.title,
        GREATEST(
            ts_rank(c.search_vector, websearch_to_tsquery('english', $2)),
            CASE WHEN similarity(c.title, $2) >= 0.3 THEN similarity(c.title, $2) ELSE 0 END,
            CASE WHEN c.course_code ILIKE $3 OR c.title ILIKE $3 OR c.description ILIKE $3 THEN 0.35 ELSE 0 END
        ) AS rank
    FROM course.courses c
    WHERE c.org_id = $1
      AND (
          c.search_vector @@ websearch_to_tsquery('english', $2)
          OR c.course_code ILIKE $3
          OR c.title ILIKE $3
          OR c.description ILIKE $3
          OR similarity(c.title, $2) >= 0.3
      )
)
SELECT
    id,
    course_code,
    title,
    rank,
    ts_headline(
        'english',
        title || ' ' || course_code,
        websearch_to_tsquery('english', $2),
        'MaxWords=12, MinWords=2, ShortWord=2'
    ) AS snippet,
    COUNT(*) OVER () AS total
FROM matched
ORDER BY rank DESC, title ASC
LIMIT $4 OFFSET $5
`, orgID, q, like, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []adminsearch.Result
	var total int64
	for rows.Next() {
		var id uuid.UUID
		var code, title, snippet string
		var rank float64
		if err := rows.Scan(&id, &code, &title, &rank, &snippet, &total); err != nil {
			return nil, 0, err
		}
		out = append(out, adminsearch.Result{
			ID:       id.String(),
			Type:     "courses",
			Title:    title,
			Subtitle: code,
			Snippet:  snippet,
			Path:     "/org-admin/courses?q=" + url.QueryEscape(title),
			Score:    rank,
		})
	}
	return out, total, rows.Err()
}

// SearchContent finds course structure items in orgID matching free text.
func SearchContent(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, q string, limit, offset int) ([]adminsearch.Result, int64, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	like := "%" + q + "%"

	rows, err := pool.Query(ctx, `
WITH matched AS (
    SELECT
        csi.id,
        csi.kind,
        csi.title,
        c.course_code,
        c.title AS course_title,
        GREATEST(
            ts_rank(csi.search_vector, websearch_to_tsquery('english', $2)),
            CASE WHEN similarity(csi.title, $2) >= 0.3 THEN similarity(csi.title, $2) ELSE 0 END,
            CASE WHEN csi.title ILIKE $3 THEN 0.35 ELSE 0 END
        ) AS rank
    FROM course.course_structure_items csi
    INNER JOIN course.courses c ON c.id = csi.course_id
    WHERE c.org_id = $1
      AND csi.archived = false
      AND csi.kind NOT IN ('module', 'heading')
      AND (
          csi.search_vector @@ websearch_to_tsquery('english', $2)
          OR csi.title ILIKE $3
          OR similarity(csi.title, $2) >= 0.3
      )
)
SELECT
    id,
    kind,
    title,
    course_code,
    course_title,
    rank,
    ts_headline(
        'english',
        title,
        websearch_to_tsquery('english', $2),
        'MaxWords=12, MinWords=2, ShortWord=2'
    ) AS snippet,
    COUNT(*) OVER () AS total
FROM matched
ORDER BY rank DESC, title ASC
LIMIT $4 OFFSET $5
`, orgID, q, like, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []adminsearch.Result
	var total int64
	for rows.Next() {
		var id uuid.UUID
		var kind, title, courseCode, courseTitle, snippet string
		var rank float64
		if err := rows.Scan(&id, &kind, &title, &courseCode, &courseTitle, &rank, &snippet, &total); err != nil {
			return nil, 0, err
		}
		subtitle := contentKindLabel(kind)
		if t := strings.TrimSpace(courseTitle); t != "" {
			subtitle = fmt.Sprintf("%s · %s · %s", subtitle, t, courseCode)
		} else {
			subtitle = fmt.Sprintf("%s · %s", subtitle, courseCode)
		}
		out = append(out, adminsearch.Result{
			ID:       fmt.Sprintf("%s:%s", courseCode, id.String()),
			Type:     "content",
			Title:    title,
			Subtitle: subtitle,
			Snippet:  snippet,
			Path:     structureItemPath(courseCode, kind, id),
			Score:    rank,
		})
	}
	return out, total, rows.Err()
}

// InsertSearchLog records a scrubbed admin search query for analytics.
func InsertSearchLog(
	ctx context.Context,
	pool *pgxpool.Pool,
	actorID, orgID uuid.UUID,
	queryScrubbed string,
	userCount, courseCount, contentCount int,
	tookMs int64,
) error {
	_, err := pool.Exec(ctx, `
INSERT INTO compliance.admin_search_log
    (actor_id, org_id, query_scrubbed, user_count, course_count, content_count, took_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, actorID, orgID, queryScrubbed, userCount, courseCount, contentCount, tookMs)
	return err
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
