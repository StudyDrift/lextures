package demographics

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/demographics"
)

// SyncFromSIS upserts demographic flags from a SIS lunch code and optional boolean flags.
// Returns true when a row was created (no prior record).
func SyncFromSIS(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, lunchCode string, ell, disability, homeless, migrant *bool) (created bool, err error) {
	prior, err := repo.GetByStudentID(ctx, pool, studentID)
	if err != nil {
		return false, err
	}
	lunch := repo.MapLunchCode(lunchCode)
	in := repo.UpsertInput{
		FreeLunch:         lunch.FreeLunch,
		ReducedLunch:      lunch.ReducedLunch,
		EllStatus:         ell,
		DisabilityStatus:  disability,
		HomelessIndicator: homeless,
		MigrantIndicator:  migrant,
		DataSource:        "sis_sync",
	}
	if _, err := repo.Upsert(ctx, pool, studentID, in); err != nil {
		return false, err
	}
	return prior == nil, nil
}
