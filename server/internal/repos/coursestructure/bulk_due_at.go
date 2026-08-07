package coursestructure

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DueAtUpdate is one bulk due-date change for a child structure item.
type DueAtUpdate struct {
	ItemID uuid.UUID
	DueAt  *time.Time
}

// BulkPatchChildDueAt applies due_at updates in a single transaction.
// Returns (updated, failed) counts. Failed means item not found / wrong kind / not a child.
func BulkPatchChildDueAt(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, updates []DueAtUpdate) (updated, failed int, err error) {
	if len(updates) == 0 {
		return 0, 0, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, u := range updates {
		tag, execErr := tx.Exec(ctx, `
			UPDATE course.course_structure_items
			SET due_at = $1, updated_at = NOW()
			WHERE id = $2
			  AND course_id = $3
			  AND parent_id IS NOT NULL
			  AND kind `+patchableChildKindsSQL,
			u.DueAt, u.ItemID, courseID,
		)
		if execErr != nil {
			return 0, 0, execErr
		}
		if tag.RowsAffected() == 0 {
			failed++
			continue
		}
		updated++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return updated, failed, nil
}
