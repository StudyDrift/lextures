package readinglevel

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	readingsvc "github.com/lextures/lextures/server/internal/service/readinglevel"
)

// ItemType identifies scorable module content.
type ItemType string

const (
	TypeContentPage ItemType = "content_page"
	TypeAssignment  ItemType = "assignment"
)

// StoredScore is persisted FKGL/FRE for an item.
type StoredScore struct {
	FKGL *float64
	FRE  *float64
}

// CachedSimplified is a row from i18n.simplified_content_cache.
type CachedSimplified struct {
	SimplifiedText string
	ComputedFKGL   *float64
	TargetFKGL     int
}

// UpdateScoreForItem writes FKGL/FRE to the appropriate module table.
func UpdateScoreForItem(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType, sc readingsvc.Score) error {
	if pool == nil {
		return errors.New("readinglevel repo: nil pool")
	}
	var fkgl, fre *float64
	if sc.Sufficient {
		f := sc.FKGL
		r := sc.FRE
		fkgl = &f
		fre = &r
	}
	switch itemType {
	case TypeContentPage:
		_, err := pool.Exec(ctx, `
UPDATE course.module_content_pages
SET reading_level_fkgl = $2, reading_level_fre = $3, updated_at = NOW()
WHERE structure_item_id = $1
`, itemID, fkgl, fre)
		return err
	case TypeAssignment:
		_, err := pool.Exec(ctx, `
UPDATE course.module_assignments
SET reading_level_fkgl = $2, reading_level_fre = $3, updated_at = NOW()
WHERE structure_item_id = $1
`, itemID, fkgl, fre)
		return err
	default:
		return errors.New("readinglevel repo: unknown item type")
	}
}

// GetScore loads stored scores for a content page or assignment.
func GetScore(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType) (StoredScore, error) {
	if pool == nil {
		return StoredScore{}, errors.New("readinglevel repo: nil pool")
	}
	var table string
	switch itemType {
	case TypeContentPage:
		table = "course.module_content_pages"
	case TypeAssignment:
		table = "course.module_assignments"
	default:
		return StoredScore{}, errors.New("readinglevel repo: unknown item type")
	}
	var fkgl, fre *float64
	q := `SELECT reading_level_fkgl, reading_level_fre FROM ` + table + ` WHERE structure_item_id = $1`
	err := pool.QueryRow(ctx, q, itemID).Scan(&fkgl, &fre)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredScore{}, nil
	}
	if err != nil {
		return StoredScore{}, err
	}
	return StoredScore{FKGL: fkgl, FRE: fre}, nil
}

// ResolveItemType returns content_page or assignment for a structure item in a course.
func ResolveItemType(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (ItemType, error) {
	var kind string
	err := pool.QueryRow(ctx, `
SELECT kind FROM course.course_structure_items
WHERE id = $1 AND course_id = $2
`, itemID, courseID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", pgx.ErrNoRows
	}
	if err != nil {
		return "", err
	}
	switch kind {
	case "content_page":
		return TypeContentPage, nil
	case "assignment":
		return TypeAssignment, nil
	default:
		return "", errors.New("readinglevel repo: item kind not scorable")
	}
}

// GetMarkdown loads markdown body for scoring or simplification.
func GetMarkdown(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType) (string, error) {
	var md string
	var q string
	switch itemType {
	case TypeContentPage:
		q = `SELECT markdown FROM course.module_content_pages WHERE structure_item_id = $1`
	case TypeAssignment:
		q = `SELECT markdown FROM course.module_assignments WHERE structure_item_id = $1`
	default:
		return "", errors.New("readinglevel repo: unknown item type")
	}
	err := pool.QueryRow(ctx, q, itemID).Scan(&md)
	return md, err
}

// UpsertSimplified stores or replaces cached simplified text.
func UpsertSimplified(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType, targetFKGL int, simplified string, computedFKGL *float64) error {
	_, err := pool.Exec(ctx, `
INSERT INTO i18n.simplified_content_cache (source_item_id, source_item_type, target_fkgl, simplified_text, computed_fkgl)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (source_item_id, source_item_type, target_fkgl)
DO UPDATE SET simplified_text = EXCLUDED.simplified_text,
              computed_fkgl = EXCLUDED.computed_fkgl,
              generated_at = NOW()
`, itemID, string(itemType), targetFKGL, simplified, computedFKGL)
	return err
}

// GetSimplified returns cached simplified content when present.
func GetSimplified(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType, targetFKGL int) (*CachedSimplified, error) {
	var c CachedSimplified
	c.TargetFKGL = targetFKGL
	err := pool.QueryRow(ctx, `
SELECT simplified_text, computed_fkgl
FROM i18n.simplified_content_cache
WHERE source_item_id = $1 AND source_item_type = $2 AND target_fkgl = $3
`, itemID, string(itemType), targetFKGL).Scan(&c.SimplifiedText, &c.ComputedFKGL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// EnrollmentReadingOverride returns the student's target FKGL for a course, if set.
func EnrollmentReadingOverride(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (*int, error) {
	var override *int
	err := pool.QueryRow(ctx, `
SELECT reading_level_override FROM course.course_enrollments
WHERE course_id = $1 AND user_id = $2
`, courseID, userID).Scan(&override)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return override, err
}

// SetEnrollmentReadingOverride updates accommodation target FKGL (nil clears).
func SetEnrollmentReadingOverride(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, override *int) error {
	_, err := pool.Exec(ctx, `
UPDATE course.course_enrollments SET reading_level_override = $2 WHERE id = $1
`, enrollmentID, override)
	return err
}

// LoadReadingLevelsForItems returns FKGL per structure item id for content pages and assignments.
func LoadReadingLevelsForItems(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, itemIDs []uuid.UUID) (map[uuid.UUID]*float64, error) {
	out := make(map[uuid.UUID]*float64)
	if len(itemIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
SELECT c.id, COALESCE(cp.reading_level_fkgl, ma.reading_level_fkgl)
FROM course.course_structure_items c
LEFT JOIN course.module_content_pages cp ON cp.structure_item_id = c.id
LEFT JOIN course.module_assignments ma ON ma.structure_item_id = c.id
WHERE c.course_id = $1 AND c.id = ANY($2)
  AND c.kind IN ('content_page', 'assignment')
`, courseID, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var fkgl *float64
		if err := rows.Scan(&id, &fkgl); err != nil {
			return nil, err
		}
		if fkgl != nil {
			v := *fkgl
			out[id] = &v
		}
	}
	return out, rows.Err()
}

// ListScorableItemIDs returns all content_page and assignment ids in a course.
func ListScorableItemIDs(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]uuid.UUID, []ItemType, error) {
	rows, err := pool.Query(ctx, `
SELECT id, kind FROM course.course_structure_items
WHERE course_id = $1 AND kind IN ('content_page', 'assignment') AND archived = false
`, courseID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	var types []ItemType
	for rows.Next() {
		var id uuid.UUID
		var kind string
		if err := rows.Scan(&id, &kind); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		if kind == "content_page" {
			types = append(types, TypeContentPage)
		} else {
			types = append(types, TypeAssignment)
		}
	}
	return ids, types, rows.Err()
}
