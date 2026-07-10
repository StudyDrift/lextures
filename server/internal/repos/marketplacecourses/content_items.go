package marketplacecourses

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ContentItemRow maps a curriculum slug to a structure item within a course.
type ContentItemRow struct {
	CourseSlug      string
	Slug            string
	StructureItemID uuid.UUID
	ContentVersion  int
	GradePolicy     *string
}

// LookupContentItem returns the structure item id for (courseSlug, slug), or nil when absent.
func LookupContentItem(ctx context.Context, tx pgx.Tx, courseSlug, slug string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT structure_item_id FROM settings.marketplace_course_items
WHERE course_slug = $1 AND slug = $2
`, courseSlug, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// UpsertContentItem records or updates the slug → structure_item_id mapping.
func UpsertContentItem(
	ctx context.Context,
	tx pgx.Tx,
	courseSlug, slug string,
	structureItemID uuid.UUID,
	contentVersion int,
	gradePolicy *string,
) error {
	_, err := tx.Exec(ctx, `
INSERT INTO settings.marketplace_course_items (course_slug, slug, structure_item_id, content_version, grade_policy, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (course_slug, slug) DO UPDATE SET
    structure_item_id = EXCLUDED.structure_item_id,
    content_version = EXCLUDED.content_version,
    grade_policy = EXCLUDED.grade_policy,
    updated_at = NOW()
`, courseSlug, slug, structureItemID, contentVersion, gradePolicy)
	return err
}

// ListContentItems returns all slug mappings for a marketplace course.
func ListContentItems(ctx context.Context, tx pgx.Tx, courseSlug string) ([]ContentItemRow, error) {
	rows, err := tx.Query(ctx, `
SELECT course_slug, slug, structure_item_id, content_version, grade_policy
FROM settings.marketplace_course_items
WHERE course_slug = $1
ORDER BY slug
`, courseSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ContentItemRow
	for rows.Next() {
		var r ContentItemRow
		if err := rows.Scan(&r.CourseSlug, &r.Slug, &r.StructureItemID, &r.ContentVersion, &r.GradePolicy); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MaxStoredContentVersion returns the highest content_version in the map for a course (0 when empty).
func MaxStoredContentVersion(ctx context.Context, tx pgx.Tx, courseSlug string) (int, error) {
	var v int
	err := tx.QueryRow(ctx, `
SELECT COALESCE(MAX(content_version), 0) FROM settings.marketplace_course_items WHERE course_slug = $1
`, courseSlug).Scan(&v)
	return v, err
}

// ItemContentVersion returns the stored content_version for a slug, or 0 when absent.
func ItemContentVersion(ctx context.Context, tx pgx.Tx, courseSlug, slug string) (int, bool, error) {
	var v int
	err := tx.QueryRow(ctx, `
SELECT content_version FROM settings.marketplace_course_items
WHERE course_slug = $1 AND slug = $2
`, courseSlug, slug).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return v, true, nil
}
