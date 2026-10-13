// Package adminsearchrepo provides org-scoped Postgres full-text search queries (plan 18.4).
//
// Text matching never hits the middle of a word (#791): short queries must match a
// whole word, longer ones a word prefix. Exact and whole-word hits rank above prefix
// hits so the 5-per-type omnisearch preview shows the best matches.
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
	m := newMatchPatterns(q)

	rows, err := pool.Query(ctx, `
WITH base AS (
    SELECT
        u.id,
        COALESCE(
            NULLIF(TRIM(COALESCE(u.display_name, '')), ''),
            TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, ''))
        ) AS name,
        TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS full_name,
        COALESCE(u.display_name, '') AS display_name,
        COALESCE(u.first_name, '') AS first_name,
        COALESCE(u.last_name, '') AS last_name,
        u.email,
        u.search_vector
    FROM "user".users u
    WHERE u.org_id = $1
      AND u.account_type <> 'system'
),
matched AS (
    SELECT
        b.id,
        b.name,
        b.email,
        GREATEST(
            ts_rank(b.search_vector, websearch_to_tsquery('english', $2)),
            CASE WHEN $5 AND similarity(b.email, $2) >= 0.3 THEN similarity(b.email, $2) ELSE 0 END,
            CASE WHEN $5 AND similarity(b.display_name, $2) >= 0.3 THEN similarity(b.display_name, $2) ELSE 0 END,
            CASE WHEN $5 AND similarity(b.full_name, $2) >= 0.3 THEN similarity(b.full_name, $2) ELSE 0 END,
            CASE
                WHEN lower(b.email) = lower($2) OR lower(b.display_name) = lower($2) OR lower(b.full_name) = lower($2) THEN 1.0
                WHEN b.email ~* $4 OR b.display_name ~* $4 OR b.full_name ~* $4 THEN 0.8
                WHEN b.email ~* $3 OR b.display_name ~* $3 OR b.full_name ~* $3 THEN 0.5
                ELSE 0
            END
        ) AS rank
    FROM base b
    WHERE
          b.search_vector @@ websearch_to_tsquery('english', $2)
          OR b.email ~* $3
          OR b.display_name ~* $3
          OR b.first_name ~* $3
          OR b.last_name ~* $3
          OR ($5 AND (
              similarity(b.email, $2) >= 0.3
              OR similarity(b.display_name, $2) >= 0.3
              OR similarity(b.full_name, $2) >= 0.3
          ))
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
LIMIT $6 OFFSET $7
`, orgID, q, m.match, m.word, m.fuzzy, limit, offset)
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
			Path:     "/org-admin/users/" + id.String(),
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
	m := newMatchPatterns(q)

	rows, err := pool.Query(ctx, `
WITH matched AS (
    SELECT
        c.id,
        c.course_code,
        c.title,
        GREATEST(
            ts_rank(c.search_vector, websearch_to_tsquery('english', $2)),
            CASE WHEN $5 AND similarity(c.title, $2) >= 0.3 THEN similarity(c.title, $2) ELSE 0 END,
            CASE
                WHEN lower(c.course_code) = lower($2) OR lower(c.title) = lower($2) THEN 1.0
                WHEN c.course_code ~* $4 OR c.title ~* $4 THEN 0.8
                WHEN c.course_code ~* $3 OR c.title ~* $3 THEN 0.5
                WHEN COALESCE(c.description, '') ~* $4 THEN 0.3
                ELSE 0
            END
        ) AS rank
    FROM course.courses c
    WHERE c.org_id = $1
      AND (
          c.search_vector @@ websearch_to_tsquery('english', $2)
          OR c.course_code ~* $3
          OR c.title ~* $3
          OR COALESCE(c.description, '') ~* $3
          OR ($5 AND similarity(c.title, $2) >= 0.3)
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
ORDER BY rank DESC, title ASC, course_code ASC
LIMIT $6 OFFSET $7
`, orgID, q, m.match, m.word, m.fuzzy, limit, offset)
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
			Path:     "/courses/" + url.PathEscape(code),
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
	m := newMatchPatterns(q)

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
            CASE WHEN $5 AND similarity(csi.title, $2) >= 0.3 THEN similarity(csi.title, $2) ELSE 0 END,
            CASE
                WHEN lower(csi.title) = lower($2) THEN 1.0
                WHEN csi.title ~* $4 THEN 0.8
                WHEN csi.title ~* $3 THEN 0.5
                ELSE 0
            END
        ) AS rank
    FROM course.course_structure_items csi
    INNER JOIN course.courses c ON c.id = csi.course_id
    WHERE c.org_id = $1
      AND csi.archived = false
      AND csi.kind NOT IN ('module', 'heading')
      AND (
          csi.search_vector @@ websearch_to_tsquery('english', $2)
          OR csi.title ~* $3
          OR ($5 AND similarity(csi.title, $2) >= 0.3)
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
LIMIT $6 OFFSET $7
`, orgID, q, m.match, m.word, m.fuzzy, limit, offset)
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
