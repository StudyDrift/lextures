// Package studenttodos persists per-learner kanban placement on the global Todos page.
package studenttodos

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidColumnIDs are weekday buckets plus a done column.
var ValidColumnIDs = map[string]struct{}{
	"mon": {}, "tue": {}, "wed": {}, "thu": {}, "fri": {}, "sat": {}, "sun": {}, "done": {},
}

// Placement is one item on the learner todo board.
type Placement struct {
	ItemKey   string
	ColumnID  string
	SortOrder int
}

// ListPlacements returns all board placements for a user.
func ListPlacements(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Placement, error) {
	rows, err := pool.Query(ctx, `
SELECT item_key, column_id, sort_order
FROM analytics.student_todo_board_placement
WHERE user_id = $1
ORDER BY column_id ASC, sort_order ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Placement
	for rows.Next() {
		var p Placement
		if err := rows.Scan(&p.ItemKey, &p.ColumnID, &p.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ReplaceBoard replaces all placements for the user.
func ReplaceBoard(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, columns map[string][]string) error {
	for col := range columns {
		if _, ok := ValidColumnIDs[col]; !ok {
			return fmt.Errorf("invalid todo column")
		}
	}
	seen := map[string]struct{}{}
	var toInsert []Placement
	for col, keys := range columns {
		for i, key := range keys {
			k := key
			if k == "" {
				return fmt.Errorf("invalid todo item key")
			}
			if _, dup := seen[k]; dup {
				return fmt.Errorf("duplicate todo item on board")
			}
			seen[k] = struct{}{}
			toInsert = append(toInsert, Placement{
				ItemKey:   k,
				ColumnID:  col,
				SortOrder: i,
			})
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM analytics.student_todo_board_placement WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, p := range toInsert {
		if _, err := tx.Exec(ctx, `
INSERT INTO analytics.student_todo_board_placement (user_id, item_key, column_id, sort_order, updated_at)
VALUES ($1, $2, $3, $4, now())
`, userID, p.ItemKey, p.ColumnID, p.SortOrder); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}