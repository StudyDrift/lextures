// Package imagealtrepo loads course markdown for alt-text coverage (plan 12.5).
package imagealtrepo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ItemCoverage is alt-text coverage for one course item.
type ItemCoverage struct {
	ItemID    uuid.UUID `json:"itemId"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"`
	WithAlt   int       `json:"withAlt"`
	Total     int       `json:"total"`
	Missing   int       `json:"missing"`
}

// ListCourseMarkdownItems returns markdown bodies for content pages and assignments.
func ListCourseMarkdownItems(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]struct {
	ItemID   uuid.UUID
	Title    string
	Kind     string
	Markdown string
}, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id, c.title, c.kind,
       COALESCE(cp.markdown, a.markdown, '') AS markdown
FROM course.course_structure_items c
LEFT JOIN course.module_content_pages cp ON cp.structure_item_id = c.id
LEFT JOIN course.module_assignments a ON a.structure_item_id = c.id
WHERE c.course_id = $1
  AND c.archived = false
  AND c.kind IN ('content_page', 'assignment')
ORDER BY c.sort_order`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ItemID   uuid.UUID
		Title    string
		Kind     string
		Markdown string
	}
	for rows.Next() {
		var row struct {
			ItemID   uuid.UUID
			Title    string
			Kind     string
			Markdown string
		}
		if err := rows.Scan(&row.ItemID, &row.Title, &row.Kind, &row.Markdown); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
