package coursestructure

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateDraftModule inserts a top-level module with published=false (plan 19.2 FR-5).
func CreateDraftModule(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, title string) (ItemRow, error) {
	row, err := CreateModule(ctx, pool, courseID, title)
	if err != nil {
		return ItemRow{}, err
	}
	if err := SetItemPublished(ctx, pool, courseID, row.ID, false); err != nil {
		return ItemRow{}, err
	}
	row.Published = false
	return row, nil
}

// SetItemPublished updates the published flag on a structure item.
func SetItemPublished(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, published bool) error {
	_, err := pool.Exec(ctx, `
UPDATE course.course_structure_items
SET published = $3, updated_at = NOW()
WHERE id = $1 AND course_id = $2
`, itemID, courseID, published)
	return err
}

// SetItemProvenance stores provenance metadata on a structure item (plan 19.9 / 19.2).
func SetItemProvenance(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, provenance json.RawMessage) error {
	_, err := pool.Exec(ctx, `
UPDATE course.course_structure_items
SET provenance = $3::jsonb, updated_at = NOW()
WHERE id = $1 AND course_id = $2
`, itemID, courseID, provenance)
	return err
}

// SetQuizProvenance stores provenance on module_quizzes.
func SetQuizProvenance(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, provenance json.RawMessage) error {
	_, err := pool.Exec(ctx, `
UPDATE course.module_quizzes q
SET provenance = $3::jsonb, updated_at = NOW()
FROM course.course_structure_items c
WHERE q.structure_item_id = c.id AND c.id = $1 AND c.course_id = $2
`, itemID, courseID, provenance)
	return err
}
