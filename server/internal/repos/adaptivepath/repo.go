package adaptivepath

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StructurePathRuleRow struct {
	ID              uuid.UUID
	StructureItemID uuid.UUID
	RuleType        string
	ConceptIDs      []uuid.UUID
	Threshold       float64
	TargetItemID    *uuid.UUID
	Priority        int16
	CreatedAt       string
}

type EnrollmentPathOverrideRow struct {
	EnrollmentID uuid.UUID
	ItemSequence []uuid.UUID
	CreatedBy    uuid.UUID
	CreatedAt    string
}

func UpsertPathOverride(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, itemSequence []uuid.UUID, createdBy uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.enrollment_path_overrides (enrollment_id, item_sequence, created_by)
VALUES ($1, $2, $3)
ON CONFLICT (enrollment_id) DO UPDATE
SET item_sequence = EXCLUDED.item_sequence, created_by = EXCLUDED.created_by, created_at = NOW()
`, enrollmentID, itemSequence, createdBy)
	return err
}
