package coursestructure

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InsertH5PUnderModule appends an h5p structure item linked to an existing package id.
func InsertH5PUnderModule(ctx context.Context, pool *pgxpool.Pool, courseID, moduleID, packageID uuid.UUID, title string) (ItemRow, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return ItemRow{}, errors.New("coursestructure: h5p title is required")
	}
	return insertModuleChild(ctx, pool, courseID, moduleID, "h5p", t, func(tx pgx.Tx, itemID uuid.UUID) error {
		tag, err := tx.Exec(ctx, `
			UPDATE content.h5p_packages SET structure_item_id = $2 WHERE id = $1 AND course_id = $3`,
			packageID, itemID, courseID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}
