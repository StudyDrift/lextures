package coursemoduleexternallinks

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpsertImportBody inserts or updates an external link URL for course import.
func UpsertImportBody(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, url string) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.module_external_links (structure_item_id, url, updated_at)
SELECT $2, $3, NOW()
FROM course.course_structure_items c
WHERE c.id = $2 AND c.course_id = $1 AND c.kind = 'external_link'
ON CONFLICT (structure_item_id) DO UPDATE SET
	url = EXCLUDED.url,
	updated_at = NOW()
`, courseID, itemID, url)
	return err
}
