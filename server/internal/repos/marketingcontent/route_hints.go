package marketingcontent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type RouteHint struct {
	ID           uuid.UUID  `json:"id"`
	RoutePrefix  string     `json:"routePrefix"`
	ArticleID    uuid.UUID  `json:"articleId"`
	ArticleTitle string     `json:"articleTitle"`
	ArticlePath  string     `json:"articlePath"`
	Position     int        `json:"position"`
	CreatedBy    *uuid.UUID `json:"createdBy"`
	CreatedAt    time.Time  `json:"createdAt"`
}

func ListRouteHints(ctx context.Context, q querier) ([]RouteHint, error) {
	rows, err := q.Query(ctx, `SELECT h.id,h.route_prefix,h.article_id,a.title,a.path,h.position,h.created_by,h.created_at
	 FROM marketing.content_route_hints h JOIN marketing.content_articles a ON a.id=h.article_id
	 ORDER BY h.route_prefix,h.position`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]RouteHint, 0)
	for rows.Next() {
		var h RouteHint
		if err := rows.Scan(&h.ID, &h.RoutePrefix, &h.ArticleID, &h.ArticleTitle, &h.ArticlePath, &h.Position, &h.CreatedBy, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func InsertRouteHint(ctx context.Context, tx pgx.Tx, routePrefix string, articleID uuid.UUID, position int, actor uuid.UUID) (*RouteHint, error) {
	var h RouteHint
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_route_hints (route_prefix,article_id,position,created_by)
	 VALUES ($1,$2,$3,$4) ON CONFLICT (route_prefix,article_id) DO UPDATE SET position=EXCLUDED.position
	 RETURNING id,route_prefix,article_id,position,created_by,created_at`, routePrefix, articleID, position, nullUUID(actor)).
		Scan(&h.ID, &h.RoutePrefix, &h.ArticleID, &h.Position, &h.CreatedBy, &h.CreatedAt)
	return &h, err
}

func DeleteRouteHint(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	tag, err := tx.Exec(ctx, `DELETE FROM marketing.content_route_hints WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ArticlesByRouteHint resolves tier (a): the explicit route-hints table. The
// longest matching prefix wins; ties break on the curated position.
func ArticlesByRouteHint(ctx context.Context, q querier, route string, limit int) ([]PublicArticle, error) {
	rows, err := q.Query(ctx, `SELECT `+publicColumns+`
	 FROM marketing.content_route_hints h
	 JOIN marketing.content_articles a ON a.id=h.article_id
	 LEFT JOIN marketing.content_categories c ON c.id=a.category_id
	 JOIN marketing.content_authors au ON au.slug=a.author_slug
	 LEFT JOIN marketing.content_authors rv ON rv.slug=a.reviewer_slug
	 WHERE a.status='published' AND a.deleted_at IS NULL AND a.published_at<=now()
	 AND $1 LIKE h.route_prefix || '%'
	 ORDER BY length(h.route_prefix) DESC, h.position ASC LIMIT $2`, route, limit)
	if err != nil {
		return nil, err
	}
	return scanPublicRows(rows)
}

// ArticlesByRelatedRoute resolves tier (b): published articles whose related_to[]
// contains a path the current route starts with.
func ArticlesByRelatedRoute(ctx context.Context, q querier, route string, limit int) ([]PublicArticle, error) {
	rows, err := q.Query(ctx, `SELECT `+publicColumns+publicJoins+`
	 WHERE a.status='published' AND a.deleted_at IS NULL AND a.published_at<=now()
	 AND EXISTS (SELECT 1 FROM unnest(a.related_to) rp WHERE rp<>'' AND $1 LIKE rp || '%')
	 ORDER BY a.published_at DESC LIMIT $2`, route, limit)
	if err != nil {
		return nil, err
	}
	return scanPublicRows(rows)
}

// ArticlesByCategoryPath resolves tier (c): the article's category platform_path
// is a prefix of the current route.
func ArticlesByCategoryPath(ctx context.Context, q querier, route string, limit int) ([]PublicArticle, error) {
	rows, err := q.Query(ctx, `SELECT `+publicColumns+publicJoins+`
	 WHERE a.status='published' AND a.deleted_at IS NULL AND a.published_at<=now()
	 AND c.platform_path IS NOT NULL AND c.platform_path<>'' AND $1 LIKE c.platform_path || '%'
	 ORDER BY length(c.platform_path) DESC, a.published_at DESC LIMIT $2`, route, limit)
	if err != nil {
		return nil, err
	}
	return scanPublicRows(rows)
}

func scanPublicRows(rows pgx.Rows) ([]PublicArticle, error) {
	defer rows.Close()
	out := make([]PublicArticle, 0)
	for rows.Next() {
		p, err := scanPublic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ViewerRoles returns the distinct role names usable for FR-5 role filtering:
// the viewer's active course-enrollment roles (e.g. "instructor", "student")
// plus any global app-role names. It is a best-effort signal, not an RBAC check.
func ViewerRoles(ctx context.Context, q querier, userID uuid.UUID) ([]string, error) {
	rows, err := q.Query(ctx, `
	 SELECT DISTINCT role FROM course.course_enrollments WHERE user_id=$1 AND active
	 UNION
	 SELECT DISTINCT ar.name FROM "user".user_app_roles uar JOIN "user".app_roles ar ON ar.id=uar.role_id WHERE uar.user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, rows.Err()
}

// LogSearchQuery aggregates a search query into the daily counter used for the
// zero-result-queries report. The query text is truncated so pathological input
// cannot bloat the table; no user identifier is ever stored here.
func LogSearchQuery(ctx context.Context, q execer, day time.Time, query, surface string, results int) error {
	if len(query) > 200 {
		query = query[:200]
	}
	_, err := q.Exec(ctx, `INSERT INTO marketing.content_search_queries (day,query,surface,hits,results)
	 VALUES ($1,$2,$3,1,$4)
	 ON CONFLICT (day,query,surface) DO UPDATE SET hits=marketing.content_search_queries.hits+1,results=EXCLUDED.results`,
		day.UTC().Truncate(24*time.Hour), query, surface, results)
	return err
}

type SearchGapRow struct {
	Query string `json:"query"`
	Hits  int    `json:"hits"`
}

// SearchGaps returns queries that returned zero results over the trailing window,
// most-frequent first.
func SearchGaps(ctx context.Context, q querier, since time.Time) ([]SearchGapRow, error) {
	rows, err := q.Query(ctx, `SELECT query,sum(hits)::int FROM marketing.content_search_queries
	 WHERE day>=$1 AND results=0 GROUP BY query ORDER BY 2 DESC,query LIMIT 200`, since.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SearchGapRow, 0)
	for rows.Next() {
		var g SearchGapRow
		if err := rows.Scan(&g.Query, &g.Hits); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
