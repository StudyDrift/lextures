package conditionalrelease

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PrerequisiteReachesModule reports whether adding (moduleID → prerequisiteModuleID) would close a cycle.
func PrerequisiteReachesModule(ctx context.Context, tx pgx.Tx, prerequisiteModuleID, moduleID uuid.UUID) (bool, error) {
	var found bool
	err := tx.QueryRow(ctx, `
WITH RECURSIVE reach AS (
    SELECT mp.prerequisite_module_id AS node, 1 AS depth
    FROM course.module_prerequisites mp
    WHERE mp.module_id = $1
    UNION ALL
    SELECT mp.prerequisite_module_id, reach.depth + 1
    FROM course.module_prerequisites mp
    INNER JOIN reach ON mp.module_id = reach.node
    WHERE reach.depth < 64
)
SELECT EXISTS (SELECT 1 FROM reach WHERE node = $2)
`, prerequisiteModuleID, moduleID).Scan(&found)
	if err != nil {
		return false, err
	}
	return found, nil
}

// InsertPrerequisiteEdge inserts one prerequisite edge inside an active transaction.
func InsertPrerequisiteEdge(ctx context.Context, tx pgx.Tx, moduleID, prerequisiteModuleID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
INSERT INTO course.module_prerequisites (module_id, prerequisite_module_id)
VALUES ($1, $2)
ON CONFLICT (module_id, prerequisite_module_id) DO NOTHING
`, moduleID, prerequisiteModuleID)
	return err
}
