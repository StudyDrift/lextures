package coursestructure

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InsertScormUnderModule appends a scorm structure item linked to an existing package id.
func InsertScormUnderModule(ctx context.Context, pool *pgxpool.Pool, courseID, moduleID, packageID uuid.UUID, title string) (ItemRow, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return ItemRow{}, errors.New("coursestructure: scorm title is required")
	}
	return insertModuleChild(ctx, pool, courseID, moduleID, "scorm", t, func(tx pgx.Tx, itemID uuid.UUID) error {
		tag, err := tx.Exec(ctx, `
			UPDATE content.scorm_packages SET structure_item_id = $2 WHERE id = $1 AND course_id = $3`,
			packageID, itemID, courseID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		_, err = tx.Exec(ctx, `
			UPDATE course.course_structure_items SET points_worth = 100 WHERE id = $1`, itemID)
		return err
	})
}
