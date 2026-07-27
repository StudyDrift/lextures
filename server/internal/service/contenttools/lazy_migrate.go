package contenttools

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
)

// LazyMigrateState upgrades a stored state document in memory. On failure the original
// is quarantined and Quarantined is true (FR-6 / FR-8). Does not persist the upgrade.
func LazyMigrateState(
	ctx context.Context,
	pool *pgxpool.Pool,
	toolID string,
	st *ctrepo.StateRow,
) (doc []byte, schemaVersion int, quarantined bool, err error) {
	if st == nil {
		return []byte(`{}`), 1, false, nil
	}
	from := st.StateSchemaVersion
	if from <= 0 {
		from = 1
	}
	table := DefaultMigrations().Get(toolID)
	target := DefaultMigrations().CurrentStateSchemaVersion(toolID)
	if from >= target {
		if pool != nil && st.ID != uuid.Nil {
			if open, _ := ctrepo.HasOpenQuarantine(ctx, pool, st.ID); open {
				return st.StateJSON, from, true, nil
			}
		}
		return st.StateJSON, from, false, nil
	}
	res := ApplyStateMigrations(table, from, st.StateJSON)
	if res.Quarantine {
		IncMigrationDocs(toolID, fmt.Sprintf("%d", from), fmt.Sprintf("%d", target), "quarantine")
		if pool != nil && st.ID != uuid.Nil {
			msg := errString(res.Error)
			if _, qerr := ctrepo.InsertQuarantine(ctx, pool, st.ID, toolID, from, target, msg, st.StateJSON); qerr != nil {
				return st.StateJSON, from, true, qerr
			}
		}
		return st.StateJSON, from, true, nil
	}
	if !res.Unchanged {
		IncMigrationDocs(toolID, fmt.Sprintf("%d", res.FromVersion), fmt.Sprintf("%d", res.ToVersion), "lazy")
	}
	return res.Doc, res.ToVersion, false, nil
}

func errString(err error) string {
	if err == nil {
		return "migration failed"
	}
	return err.Error()
}
