package coursestructure

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InsertVibeActivityUnderModule appends a vibe_activity structure item and a row in module_vibe_activities.
func InsertVibeActivityUnderModule(ctx context.Context, pool *pgxpool.Pool, courseID, moduleID uuid.UUID, title, html string) (ItemRow, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return ItemRow{}, errors.New("coursestructure: vibe activity title is required")
	}
	h := strings.TrimSpace(html)
	// html may legitimately be empty at creation time (populated via PATCH or the create payload)

	return insertModuleChild(ctx, pool, courseID, moduleID, "vibe_activity", t, func(tx pgx.Tx, itemID uuid.UUID) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO course.module_vibe_activities (structure_item_id, html_content)
			VALUES ($1, $2)
		`, itemID, h)
		return err
	})
}
