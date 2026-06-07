package coursestructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const reorderOffset = 10_000_000

// ErrInvalidReorder is returned when module or child ids do not match the current structure.
var ErrInvalidReorder = errors.New("coursestructure: invalid reorder")

// ApplyModuleAndChildOrder reassigns sort_order for top-level modules and each module's children.
// moduleIDsInOrder must list every non-archived top-level module id. For each such module,
// childrenByModule must list every non-archived child id in the desired order; modules with no
// children use an empty slice (or may be omitted from the map). Archived modules and children
// are unchanged.
func ApplyModuleAndChildOrder(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	moduleIDsInOrder []uuid.UUID,
	childrenByModule map[uuid.UUID][]uuid.UUID,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var lockedCourse uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM course.courses WHERE id = $1 FOR UPDATE`, courseID).Scan(&lockedCourse)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidReorder
	}
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, archived
		FROM course.course_structure_items
		WHERE course_id = $1 AND parent_id IS NULL AND kind = 'module'
		ORDER BY sort_order
	`, courseID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var visibleModules []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		var archived bool
		if err := rows.Scan(&id, &archived); err != nil {
			return err
		}
		if !archived {
			visibleModules = append(visibleModules, id)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	visibleModSet := make(map[uuid.UUID]struct{}, len(visibleModules))
	for _, id := range visibleModules {
		visibleModSet[id] = struct{}{}
	}
	orderSet := make(map[uuid.UUID]struct{}, len(moduleIDsInOrder))
	for _, id := range moduleIDsInOrder {
		orderSet[id] = struct{}{}
	}
	if len(visibleModSet) != len(orderSet) {
		return ErrInvalidReorder
	}
	for id := range visibleModSet {
		if _, ok := orderSet[id]; !ok {
			return ErrInvalidReorder
		}
	}

	for _, mid := range visibleModules {
		childRows, err := tx.Query(ctx, `
			SELECT id, archived
			FROM course.course_structure_items
			WHERE parent_id = $1
			ORDER BY sort_order
		`, mid)
		if err != nil {
			return err
		}
		var visibleChildIDs []uuid.UUID
		for childRows.Next() {
			var id uuid.UUID
			var archived bool
			if err := childRows.Scan(&id, &archived); err != nil {
				childRows.Close()
				return err
			}
			if !archived {
				visibleChildIDs = append(visibleChildIDs, id)
			}
		}
		childRows.Close()
		if err := childRows.Err(); err != nil {
			return err
		}

		visibleChildSet := make(map[uuid.UUID]struct{}, len(visibleChildIDs))
		for _, id := range visibleChildIDs {
			visibleChildSet[id] = struct{}{}
		}
		specified := childrenByModule[mid]
		if specified == nil {
			specified = []uuid.UUID{}
		}
		specSet := make(map[uuid.UUID]struct{}, len(specified))
		for _, id := range specified {
			specSet[id] = struct{}{}
		}
		if len(visibleChildSet) != len(specSet) {
			return ErrInvalidReorder
		}
		for id := range visibleChildSet {
			if _, ok := specSet[id]; !ok {
				return ErrInvalidReorder
			}
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE course.course_structure_items
		SET sort_order = sort_order + $2
		WHERE course_id = $1 AND parent_id IS NULL AND kind = 'module'
	`, courseID, reorderOffset); err != nil {
		return err
	}

	for ord, id := range moduleIDsInOrder {
		if _, err := tx.Exec(ctx, `
			UPDATE course.course_structure_items
			SET sort_order = $3
			WHERE id = $1 AND course_id = $2 AND parent_id IS NULL AND kind = 'module'
		`, id, courseID, ord); err != nil {
			return err
		}
	}

	for _, mid := range visibleModules {
		childIDs := childrenByModule[mid]
		if childIDs == nil {
			childIDs = []uuid.UUID{}
		}
		if len(childIDs) == 0 {
			continue
		}

		if _, err := tx.Exec(ctx, `
			UPDATE course.course_structure_items
			SET sort_order = sort_order + $2
			WHERE parent_id = $1
		`, mid, reorderOffset); err != nil {
			return err
		}

		for ord, cid := range childIDs {
			if _, err := tx.Exec(ctx, `
				UPDATE course.course_structure_items
				SET sort_order = $3
				WHERE id = $1 AND parent_id = $2
			`, cid, mid, ord); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}
