package adaptivecontent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ServingRow is a course.adaptation_servings row (AC.1 / AC.6).
type ServingRow struct {
	ID                 uuid.UUID
	UnitID             uuid.UUID
	EnrollmentID       uuid.UUID
	ProfileID          *uuid.UUID
	VariantID          *uuid.UUID
	WasHoldout         bool
	WasFallback        bool
	ServedAt           time.Time
	ContentVersion     int32
	ViewCount          int32
	FirstViewedAt      time.Time
	ViewOriginalClicks int32
}

// ServingUpsert is the input for recording a content exposure.
type ServingUpsert struct {
	UnitID         uuid.UUID
	EnrollmentID   uuid.UUID
	ProfileID      *uuid.UUID
	VariantID      *uuid.UUID
	WasHoldout     bool
	WasFallback    bool
	ContentVersion int32
}

// UpsertServing records or updates a serving exposure for (unit, enrollment, content_version).
// Re-opens increment view_count and refresh served_at; first_viewed_at is preserved.
func UpsertServing(ctx context.Context, pool *pgxpool.Pool, in ServingUpsert) (*ServingRow, error) {
	if in.ContentVersion <= 0 {
		in.ContentVersion = 1
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.adaptation_servings (
  unit_id, enrollment_id, profile_id, variant_id, was_holdout, was_fallback,
  content_version, view_count, first_viewed_at, served_at, view_original_clicks
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, 1, NOW(), NOW(), 0
)
ON CONFLICT (unit_id, enrollment_id, content_version) DO UPDATE SET
  profile_id = EXCLUDED.profile_id,
  variant_id = EXCLUDED.variant_id,
  was_holdout = EXCLUDED.was_holdout,
  was_fallback = EXCLUDED.was_fallback,
  view_count = course.adaptation_servings.view_count + 1,
  served_at = NOW()
RETURNING id, unit_id, enrollment_id, profile_id, variant_id, was_holdout, was_fallback,
          served_at, content_version, view_count, first_viewed_at, view_original_clicks
`, in.UnitID, in.EnrollmentID, in.ProfileID, in.VariantID, in.WasHoldout, in.WasFallback, in.ContentVersion)
	return scanServing(row)
}

// IncrementViewOriginalClicks increments the counter for an exposure. Returns the new count or 0 if no row.
func IncrementViewOriginalClicks(
	ctx context.Context,
	pool *pgxpool.Pool,
	unitID, enrollmentID uuid.UUID,
	contentVersion int32,
) (int32, error) {
	if contentVersion <= 0 {
		contentVersion = 1
	}
	var n int32
	err := pool.QueryRow(ctx, `
UPDATE course.adaptation_servings
SET view_original_clicks = view_original_clicks + 1
WHERE unit_id = $1 AND enrollment_id = $2 AND content_version = $3
RETURNING view_original_clicks
`, unitID, enrollmentID, contentVersion).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return n, err
}

// GetServing returns the serving row for an exposure, or nil.
func GetServing(
	ctx context.Context,
	pool *pgxpool.Pool,
	unitID, enrollmentID uuid.UUID,
	contentVersion int32,
) (*ServingRow, error) {
	if contentVersion <= 0 {
		contentVersion = 1
	}
	row := pool.QueryRow(ctx, `
SELECT id, unit_id, enrollment_id, profile_id, variant_id, was_holdout, was_fallback,
       served_at, content_version, view_count, first_viewed_at, view_original_clicks
FROM course.adaptation_servings
WHERE unit_id = $1 AND enrollment_id = $2 AND content_version = $3
`, unitID, enrollmentID, contentVersion)
	s, err := scanServing(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func scanServing(row scannable) (*ServingRow, error) {
	var s ServingRow
	err := row.Scan(
		&s.ID, &s.UnitID, &s.EnrollmentID, &s.ProfileID, &s.VariantID,
		&s.WasHoldout, &s.WasFallback, &s.ServedAt, &s.ContentVersion,
		&s.ViewCount, &s.FirstViewedAt, &s.ViewOriginalClicks,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetActiveUnitByBaseContentItem returns the active unit whose base_content_item_id matches, or nil.
// When multiple units share a base item, prefers the most recently updated active unit.
func GetActiveUnitByBaseContentItem(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, baseContentItemID uuid.UUID,
) (*UnitRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, course_id, target_kind, target_module_item_id, target_outcome_id,
       base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
       allowed_axes, status, created_by, created_at, updated_at,
       trigger_mode, mastery_freshness_days, content_version, min_fidelity,
       quarantined, quarantined_reason
FROM course.adaptive_content_units
WHERE course_id = $1 AND base_content_item_id = $2 AND status = 'active'
ORDER BY updated_at DESC
LIMIT 1
`, courseID, baseContentItemID)
	u, err := scanUnit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetServableVariant returns an approved or auto_served variant for (unit, signature, version), or nil.
func GetServableVariant(
	ctx context.Context,
	pool *pgxpool.Pool,
	unitID uuid.UUID,
	signature string,
	contentVersion int32,
) (*VariantRow, error) {
	if contentVersion <= 0 {
		contentVersion = 1
	}
	row := pool.QueryRow(ctx, `
SELECT id, unit_id, profile_signature, axes_applied, variant_markdown, model,
       fidelity_score, safety_flags, status, approved_by, created_at,
       prompt_version, content_version, prompt_tokens, completion_tokens, a11y_flags,
       human_edited, reviewed_by, reviewed_at, review_note, variant_version
FROM course.content_variants
WHERE unit_id = $1 AND profile_signature = $2
  AND content_version = $3
  AND status IN ('approved', 'auto_served')
LIMIT 1
`, unitID, signature, contentVersion)
	v, err := scanVariant(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
