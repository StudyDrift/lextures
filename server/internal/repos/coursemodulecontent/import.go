package coursemodulecontent

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpsertImportBody inserts or updates a content page body for course import.
func UpsertImportBody(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, markdown string) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.module_content_pages (structure_item_id, markdown, updated_at)
SELECT c.id, $3, NOW()
FROM course.course_structure_items c
WHERE c.id = $1 AND c.course_id = $2 AND c.kind = 'content_page'
ON CONFLICT (structure_item_id) DO UPDATE
SET markdown = EXCLUDED.markdown, updated_at = NOW()
`, itemID, courseID, markdown)
	return err
}
