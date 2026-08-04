package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StampIntegritySettingsReviewed sets integrity_settings_reviewed_at = NOW() for a course.
// Drives checklist item integrity.high-stakes-settings (CC.5).
// Accommodations review is stamped inline from studentaccommodations writes (same table)
// to avoid an import cycle with this package.
func StampIntegritySettingsReviewed(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE course.courses
SET integrity_settings_reviewed_at = NOW(), updated_at = NOW()
WHERE id = $1
`, courseID)
	return err
}
