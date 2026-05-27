package coursestructure

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VibeActivityRow is the supporting data for a vibe_activity structure item.
type VibeActivityRow struct {
	HTML string
}

// GetVibeActivity loads the HTML content for a given structure item (caller must have already verified course + kind).
func GetVibeActivity(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID) (VibeActivityRow, error) {
	var r VibeActivityRow
	err := pool.QueryRow(ctx, `
		SELECT html_content FROM course.module_vibe_activities WHERE structure_item_id = $1
	`, itemID).Scan(&r.HTML)
	if err != nil {
		return VibeActivityRow{}, err
	}
	return r, nil
}

// UpdateVibeActivityHTML updates only the HTML content (and timestamp).
func UpdateVibeActivityHTML(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, html string) error {
	_, err := pool.Exec(ctx, `
		UPDATE course.module_vibe_activities
		SET html_content = $2, updated_at = NOW()
		WHERE structure_item_id = $1
	`, itemID, html)
	return err
}
