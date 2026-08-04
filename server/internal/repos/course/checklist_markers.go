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

// StampA11yReviewed sets a11y_reviewed_at = NOW() for a course (CC.6).
func StampA11yReviewed(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE course.courses
SET a11y_reviewed_at = NOW(), updated_at = NOW()
WHERE id = $1
`, courseID)
	return err
}

// StampStudentPreview sets student_preview_at = NOW() when staff use View as: Student (CC.6).
func StampStudentPreview(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE course.courses
SET student_preview_at = NOW(), updated_at = NOW()
WHERE id = $1
`, courseID)
	return err
}

// StampLastExport sets last_export_at = NOW() after a successful course export (CC.6).
func StampLastExport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE course.courses
SET last_export_at = NOW(), updated_at = NOW()
WHERE id = $1
`, courseID)
	return err
}

// StampA11yReviewedByCourseCode resolves the course then stamps a11y_reviewed_at.
func StampA11yReviewedByCourseCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) error {
	id, err := GetIDByCourseCode(ctx, pool, courseCode)
	if err != nil || id == nil {
		return err
	}
	return StampA11yReviewed(ctx, pool, *id)
}

// StampStudentPreviewByCourseCode resolves the course then stamps student_preview_at.
func StampStudentPreviewByCourseCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) error {
	id, err := GetIDByCourseCode(ctx, pool, courseCode)
	if err != nil || id == nil {
		return err
	}
	return StampStudentPreview(ctx, pool, *id)
}

// StampLastExportByCourseCode resolves the course then stamps last_export_at.
func StampLastExportByCourseCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) error {
	id, err := GetIDByCourseCode(ctx, pool, courseCode)
	if err != nil || id == nil {
		return err
	}
	return StampLastExport(ctx, pool, *id)
}
