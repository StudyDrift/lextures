package accessibility

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/lextures/lextures/server/internal/repos/accessibilityprofiles"
	stac "github.com/lextures/lextures/server/internal/repos/studentaccommodations"
)

// Apply propagates an active profile to the 2.11 override engine by creating a global
// (course_id NULL) student_accommodations row for the student, which the quiz delivery
// layer already resolves for every enrolled course (AC-1). It records the created row id
// back on the profile so deactivation can remove exactly that override (AC-5).
//
// coordinatorID is stamped as created_by on the override row.
func Apply(
	ctx context.Context, pool *pgxpool.Pool,
	profileID, studentID uuid.UUID,
	types []string, params json.RawMessage,
	effectiveFrom, effectiveUntil *time.Time,
	coordinatorID uuid.UUID,
) (*stac.Row, error) {
	w := BuildWrite(types, params)
	w.EffectiveFrom = effectiveFrom
	w.EffectiveUntil = effectiveUntil

	row, err := stac.InsertRow(ctx, pool, studentID, nil, w, coordinatorID)
	if err != nil {
		return nil, err
	}
	if err := repo.SetApplied(ctx, pool, profileID, &row.ID); err != nil {
		return nil, err
	}
	for _, t := range types {
		RecordApplied(t)
	}
	return row, nil
}

// Deactivate removes the propagated override for a profile (soft-deletes the
// student_accommodations row), so standard limits resume on the next attempt (AC-5).
func Deactivate(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, appliedID *uuid.UUID) error {
	if appliedID == nil {
		return nil
	}
	_, err := stac.DeleteRow(ctx, pool, *appliedID, studentID)
	return err
}
