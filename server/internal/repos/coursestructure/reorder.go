package coursestructure

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// childTempBase is far above any legitimate dense sort_order (0..n). Children are
// parked here with course-wide unique values before reparent / final assignment so
// idx_course_structure_items_child_order cannot fire mid-update.
const childTempBase = 1_000_000_000

// ErrInvalidReorder is returned when module or child ids do not match the current structure.
var ErrInvalidReorder = errors.New("coursestructure: invalid reorder")

// ApplyModuleAndChildOrder reassigns sort_order for top-level modules and each module's children.
// It also reparents children when the same child id appears under a different module than before
// (move item between modules). moduleIDsInOrder must list every non-archived top-level module id.
// The union of childrenByModule must equal every non-archived child id in the course (each child
// appears under exactly one module). Modules with no children use an empty slice (or may be
// omitted from the map). Archived modules and children are unchanged in membership; their
// sort_order may be compacted to keep unique indexes free of collisions.
//
// Implementation notes (idx_course_structure_items_child_order / top_level_order):
// PostgreSQL checks non-deferrable unique indexes per row as each row is updated. Reordering
// therefore parks every affected row on temporary sort_orders that cannot collide with final
// 0..n values (and, for children, are unique across the whole course so reparent is safe),
// then writes final orders and compacts archived / leftover rows.
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

	// Current non-archived children: childID → parent moduleID.
	currentChildParent := make(map[uuid.UUID]uuid.UUID)
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
		for childRows.Next() {
			var id uuid.UUID
			var archived bool
			if err := childRows.Scan(&id, &archived); err != nil {
				childRows.Close()
				return err
			}
			if !archived {
				currentChildParent[id] = mid
			}
		}
		childRows.Close()
		if err := childRows.Err(); err != nil {
			return err
		}
	}

	// Requested placement: childID → new parent moduleID. Each child may appear once.
	requestedChildParent := make(map[uuid.UUID]uuid.UUID)
	for mid, kids := range childrenByModule {
		if _, ok := visibleModSet[mid]; !ok {
			return ErrInvalidReorder
		}
		for _, cid := range kids {
			if _, dup := requestedChildParent[cid]; dup {
				return ErrInvalidReorder
			}
			requestedChildParent[cid] = mid
		}
	}
	// Modules omitted from the map are treated as empty (no children requested).

	if len(requestedChildParent) != len(currentChildParent) {
		return ErrInvalidReorder
	}
	for cid := range currentChildParent {
		if _, ok := requestedChildParent[cid]; !ok {
			return ErrInvalidReorder
		}
	}

	// --- Top-level (unique index: course_id, sort_order WHERE parent_id IS NULL) ---
	// 1) Mirror onto negatives (no collision with non-negative sources).
	// 2) Assign final module order 0..n-1.
	// 3) Compact remaining top-level rows after the module sequence.
	if _, err := tx.Exec(ctx, `
		UPDATE course.course_structure_items
		SET sort_order = -1 - sort_order
		WHERE course_id = $1 AND parent_id IS NULL AND sort_order >= 0
	`, courseID); err != nil {
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

	if _, err := tx.Exec(ctx, `
		WITH remaining AS (
			SELECT id,
				$2::int + (ROW_NUMBER() OVER (ORDER BY sort_order, id) - 1)::int AS new_ord
			FROM course.course_structure_items
			WHERE course_id = $1
			  AND parent_id IS NULL
			  AND NOT (kind = 'module' AND NOT archived)
		)
		UPDATE course.course_structure_items AS c
		SET sort_order = remaining.new_ord
		FROM remaining
		WHERE c.id = remaining.id
	`, courseID, len(moduleIDsInOrder)); err != nil {
		return err
	}

	// --- Children (unique index: parent_id, sort_order WHERE parent_id IS NOT NULL) ---
	// 1) Mirror onto negatives (safe vs current non-negative values).
	// 2) Park every child on a course-wide unique high temp so reparent cannot collide.
	// 3) Reparent live children to their target modules (temps unchanged).
	// 4) Write final dense sort_orders 0..n-1 per module.
	// 5) Compact archived (and any other leftovers) after the live sequence.
	// 6) Compact children under non-visible parents (e.g. archived modules).
	if _, err := tx.Exec(ctx, `
		UPDATE course.course_structure_items
		SET sort_order = -1 - sort_order
		WHERE course_id = $1 AND parent_id IS NOT NULL AND sort_order >= 0
	`, courseID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		WITH numbered AS (
			SELECT id,
				($2::int + ROW_NUMBER() OVER (ORDER BY parent_id, id))::int AS tmp_ord
			FROM course.course_structure_items
			WHERE course_id = $1 AND parent_id IS NOT NULL
		)
		UPDATE course.course_structure_items AS c
		SET sort_order = numbered.tmp_ord
		FROM numbered
		WHERE c.id = numbered.id
	`, courseID, childTempBase); err != nil {
		return err
	}

	for _, mid := range visibleModules {
		childIDs := childrenByModule[mid]
		if childIDs == nil {
			childIDs = []uuid.UUID{}
		}
		for _, cid := range childIDs {
			tag, err := tx.Exec(ctx, `
				UPDATE course.course_structure_items
				SET parent_id = $2
				WHERE id = $1 AND course_id = $3 AND parent_id IS NOT NULL AND NOT archived
			`, cid, mid, courseID)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return ErrInvalidReorder
			}
		}
	}

	for _, mid := range visibleModules {
		childIDs := childrenByModule[mid]
		if childIDs == nil {
			childIDs = []uuid.UUID{}
		}
		for ord, cid := range childIDs {
			tag, err := tx.Exec(ctx, `
				UPDATE course.course_structure_items
				SET sort_order = $3
				WHERE id = $1 AND course_id = $2 AND parent_id = $4 AND NOT archived
			`, cid, courseID, ord, mid)
			if err != nil {
				return err
			}
			if tag.RowsAffected() != 1 {
				return ErrInvalidReorder
			}
		}
		// Archived children (and any unexpected leftovers) after the live sequence.
		if _, err := tx.Exec(ctx, `
			WITH leftovers AS (
				SELECT id,
					$2::int + (ROW_NUMBER() OVER (ORDER BY sort_order, id) - 1)::int AS new_ord
				FROM course.course_structure_items
				WHERE parent_id = $1 AND archived
			)
			UPDATE course.course_structure_items AS c
			SET sort_order = leftovers.new_ord
			FROM leftovers
			WHERE c.id = leftovers.id
		`, mid, len(childIDs)); err != nil {
			return err
		}
	}

	if len(visibleModules) == 0 {
		visibleModules = []uuid.UUID{}
	}
	if _, err := tx.Exec(ctx, `
		WITH parents AS (
			SELECT DISTINCT parent_id
			FROM course.course_structure_items
			WHERE course_id = $1
			  AND parent_id IS NOT NULL
			  AND NOT (parent_id = ANY($2::uuid[]))
		),
		numbered AS (
			SELECT c.id,
				(ROW_NUMBER() OVER (PARTITION BY c.parent_id ORDER BY c.sort_order, c.id) - 1)::int AS new_ord
			FROM course.course_structure_items c
			INNER JOIN parents p ON p.parent_id = c.parent_id
		)
		UPDATE course.course_structure_items AS c
		SET sort_order = numbered.new_ord
		FROM numbered
		WHERE c.id = numbered.id
	`, courseID, visibleModules); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
