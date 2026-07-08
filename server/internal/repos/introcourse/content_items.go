package introcourse

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ContentItemRow maps a curriculum slug to a structure item.
type ContentItemRow struct {
	Slug            string
	StructureItemID uuid.UUID
	ContentVersion  int
	GradePolicy     *string
}

// LookupContentItem returns the structure item id for slug, or nil when absent.
func LookupContentItem(ctx context.Context, tx pgx.Tx, slug string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT structure_item_id FROM settings.intro_course_items WHERE slug = $1
`, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// UpsertContentItem records or updates the slug → structure_item_id mapping.
func UpsertContentItem(ctx context.Context, tx pgx.Tx, slug string, structureItemID uuid.UUID, contentVersion int, gradePolicy *string) error {
	_, err := tx.Exec(ctx, `
INSERT INTO settings.intro_course_items (slug, structure_item_id, content_version, grade_policy, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (slug) DO UPDATE SET
    structure_item_id = EXCLUDED.structure_item_id,
    content_version = EXCLUDED.content_version,
    grade_policy = EXCLUDED.grade_policy,
    updated_at = NOW()
`, slug, structureItemID, contentVersion, gradePolicy)
	return err
}

// ListContentItems returns all slug mappings for the intro course curriculum.
func ListContentItems(ctx context.Context, tx pgx.Tx) ([]ContentItemRow, error) {
	rows, err := tx.Query(ctx, `
SELECT slug, structure_item_id, content_version, grade_policy
FROM settings.intro_course_items
ORDER BY slug
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ContentItemRow
	for rows.Next() {
		var r ContentItemRow
		if err := rows.Scan(&r.Slug, &r.StructureItemID, &r.ContentVersion, &r.GradePolicy); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LookupGradePolicy returns the stored grade_policy for slug, or empty when unset.
func LookupGradePolicy(ctx context.Context, tx pgx.Tx, slug string) (string, error) {
	var policy *string
	err := tx.QueryRow(ctx, `
SELECT grade_policy FROM settings.intro_course_items WHERE slug = $1
`, slug).Scan(&policy)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if policy == nil {
		return "", nil
	}
	return *policy, nil
}

// AssignmentGroupIDByName resolves an assignment group id by display name.
func AssignmentGroupIDByName(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, name string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM course.assignment_groups WHERE course_id = $1 AND name = $2
`, courseID, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// MaxStoredContentVersion returns the highest content_version in the map (0 when empty).
func MaxStoredContentVersion(ctx context.Context, tx pgx.Tx) (int, error) {
	var v int
	err := tx.QueryRow(ctx, `
SELECT COALESCE(MAX(content_version), 0) FROM settings.intro_course_items
`).Scan(&v)
	return v, err
}