package introcourse

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// StatusRow is the singleton settings.intro_course_status record.
type StatusRow struct {
	ContentVersion         int
	LastSyncedAt           *time.Time
	LastSyncResult         *string
	LastValidatedAt        *time.Time
	LastValidationResult   *string
	UpdatedAt              time.Time
}

// LoadStatus returns the singleton status row (zero values when absent).
func LoadStatus(ctx context.Context, exec interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (StatusRow, error) {
	var row StatusRow
	err := exec.QueryRow(ctx, `
SELECT content_version, last_synced_at, last_sync_result,
       last_validated_at, last_validation_result, updated_at
FROM settings.intro_course_status
WHERE id = TRUE
`).Scan(
		&row.ContentVersion, &row.LastSyncedAt, &row.LastSyncResult,
		&row.LastValidatedAt, &row.LastValidationResult, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return StatusRow{}, nil
	}
	return row, err
}

// RecordSyncResult persists the outcome of a content sync run.
func RecordSyncResult(ctx context.Context, exec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, contentVersion int, at time.Time, result string) error {
	_, err := exec.Exec(ctx, `
INSERT INTO settings.intro_course_status (id, content_version, last_synced_at, last_sync_result, updated_at)
VALUES (TRUE, $1, $2, $3, NOW())
ON CONFLICT (id) DO UPDATE SET
    content_version = EXCLUDED.content_version,
    last_synced_at = EXCLUDED.last_synced_at,
    last_sync_result = EXCLUDED.last_sync_result,
    updated_at = NOW()
`, contentVersion, at, result)
	return err
}

// RecordValidationResult persists the outcome of fixture validation.
func RecordValidationResult(ctx context.Context, exec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, at time.Time, result string) error {
	_, err := exec.Exec(ctx, `
INSERT INTO settings.intro_course_status (id, last_validated_at, last_validation_result, updated_at)
VALUES (TRUE, $1, $2, NOW())
ON CONFLICT (id) DO UPDATE SET
    last_validated_at = EXCLUDED.last_validated_at,
    last_validation_result = EXCLUDED.last_validation_result,
    updated_at = NOW()
`, at, result)
	return err
}

// SlugByStructureItemID returns the intro_course_items slug for a structure item.
func SlugByStructureItemID(ctx context.Context, exec interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, itemID uuid.UUID) (string, error) {
	var slug string
	err := exec.QueryRow(ctx, `
SELECT slug FROM settings.intro_course_items WHERE structure_item_id = $1
`, itemID).Scan(&slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return slug, err
}

// CountModules returns published, non-archived intro course modules.
func CountModules(ctx context.Context, exec interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, courseID uuid.UUID) (int, error) {
	var n int
	err := exec.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM settings.intro_course_items ici
INNER JOIN course.course_structure_items csi ON csi.id = ici.structure_item_id
WHERE csi.course_id = $1
  AND csi.kind = 'module'
  AND csi.published
  AND NOT csi.archived
`, courseID).Scan(&n)
	return n, err
}

// AvgCompletionHours returns mean hours from enrollment to completion for completers.
func AvgCompletionHours(ctx context.Context, exec interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, courseID uuid.UUID) (*float64, error) {
	var avg *float64
	err := exec.QueryRow(ctx, `
SELECT AVG(EXTRACT(EPOCH FROM (c.completed_at - ce.created_at)) / 3600.0)
FROM settings.intro_course_completions c
INNER JOIN course.course_enrollments ce
    ON ce.user_id = c.user_id
   AND ce.course_id = $1
   AND ce.role = 'student'
   AND ce.active
   AND ce.state = 'active'
`, courseID).Scan(&avg)
	return avg, err
}