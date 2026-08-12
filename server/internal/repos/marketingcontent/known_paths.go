package marketingcontent

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func KnownPaths(ctx context.Context, q querier) (map[string]struct{}, error) {
	rows, err := q.Query(ctx, `SELECT path FROM marketing.content_known_paths UNION SELECT path FROM marketing.content_articles WHERE status='published' AND deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out[path] = struct{}{}
	}
	return out, rows.Err()
}

func ReplaceStaticKnownPaths(ctx context.Context, tx pgx.Tx, paths []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM marketing.content_known_paths WHERE source='static_route'`); err != nil {
		return err
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#") {
			return fmt.Errorf("invalid static route %q", path)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO marketing.content_known_paths(path,source) VALUES($1,'static_route') ON CONFLICT(path) DO UPDATE SET source=CASE WHEN marketing.content_known_paths.source='article' THEN 'article' ELSE 'static_route' END,updated_at=now()`, path); err != nil {
			return err
		}
	}
	return nil
}
