package adaptivecontent

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsRow is a course.adaptive_content_settings row.
type SettingsRow struct {
	CourseID                  uuid.UUID
	AllowedAxes               []string
	DefaultStrategy           string
	HoldoutPercent            int16
	MonthlyTokenBudget        int64
	RequireInstructorApproval bool
	StudentOptoutAllowed      bool
	UpdatedBy                 *uuid.UUID
	UpdatedAt                 time.Time
	// AC.4
	GenerationPaused   bool
	MaxPrewarmVariants int16
	BudgetPeriodStart  *time.Time // date only; stored as DATE
	TokensUsedPeriod   int64
}

// UnitRow is a course.adaptive_content_units row.
type UnitRow struct {
	ID                   uuid.UUID
	CourseID             uuid.UUID
	TargetKind           string
	TargetModuleItemID   *uuid.UUID
	TargetOutcomeID      *uuid.UUID
	BaseContentItemID    uuid.UUID
	PreAssessmentItemID  *uuid.UUID
	PostAssessmentItemID *uuid.UUID
	AllowedAxes          []string
	Status               string
	CreatedBy            uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
	// AC.2
	TriggerMode          string
	MasteryFreshnessDays int16
	// AC.3
	ContentVersion int32
	MinFidelity    float64
	// AC.8
	Quarantined       bool
	QuarantinedReason *string
}

// DefaultSettings returns in-memory defaults matching the migration defaults.
func DefaultSettings(courseID uuid.UUID) SettingsRow {
	return SettingsRow{
		CourseID:                  courseID,
		AllowedAxes:               []string{"emphasis", "scaffolding", "reading_level", "misconception"},
		DefaultStrategy:           "balanced",
		HoldoutPercent:            0,
		MonthlyTokenBudget:        0,
		RequireInstructorApproval: false,
		StudentOptoutAllowed:      true,
		UpdatedAt:                 time.Now().UTC(),
		GenerationPaused:          false,
		MaxPrewarmVariants:        12,
		TokensUsedPeriod:          0,
	}
}

// GetSettings returns settings for a course, or nil if no row exists.
func GetSettings(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*SettingsRow, error) {
	var r SettingsRow
	var updatedBy *uuid.UUID
	var periodStart *time.Time
	err := pool.QueryRow(ctx, `
SELECT course_id, allowed_axes, default_strategy, holdout_percent, monthly_token_budget,
       require_instructor_approval, student_optout_allowed, updated_by, updated_at,
       generation_paused, max_prewarm_variants, budget_period_start, tokens_used_period
FROM course.adaptive_content_settings
WHERE course_id = $1
`, courseID).Scan(
		&r.CourseID, &r.AllowedAxes, &r.DefaultStrategy, &r.HoldoutPercent, &r.MonthlyTokenBudget,
		&r.RequireInstructorApproval, &r.StudentOptoutAllowed, &updatedBy, &r.UpdatedAt,
		&r.GenerationPaused, &r.MaxPrewarmVariants, &periodStart, &r.TokensUsedPeriod,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.UpdatedBy = updatedBy
	r.BudgetPeriodStart = periodStart
	if r.AllowedAxes == nil {
		r.AllowedAxes = []string{}
	}
	if r.MaxPrewarmVariants <= 0 {
		r.MaxPrewarmVariants = 12
	}
	return &r, nil
}

// UpsertSettings inserts or updates settings for a course.
// AC.4 pipeline fields (generation_paused, max_prewarm_variants) are preserved on update when
// the caller leaves MaxPrewarmVariants at 0 (meaning "keep existing / default on insert").
func UpsertSettings(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, s SettingsRow, actorUserID uuid.UUID) (*SettingsRow, error) {
	maxPrewarm := s.MaxPrewarmVariants
	if maxPrewarm <= 0 {
		maxPrewarm = 12
	}
	var r SettingsRow
	var updatedBy *uuid.UUID
	var periodStart *time.Time
	err := pool.QueryRow(ctx, `
INSERT INTO course.adaptive_content_settings (
  course_id, allowed_axes, default_strategy, holdout_percent, monthly_token_budget,
  require_instructor_approval, student_optout_allowed, updated_by, updated_at,
  generation_paused, max_prewarm_variants
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9, $10)
ON CONFLICT (course_id) DO UPDATE SET
  allowed_axes = EXCLUDED.allowed_axes,
  default_strategy = EXCLUDED.default_strategy,
  holdout_percent = EXCLUDED.holdout_percent,
  monthly_token_budget = EXCLUDED.monthly_token_budget,
  require_instructor_approval = EXCLUDED.require_instructor_approval,
  student_optout_allowed = EXCLUDED.student_optout_allowed,
  generation_paused = EXCLUDED.generation_paused,
  max_prewarm_variants = EXCLUDED.max_prewarm_variants,
  updated_by = EXCLUDED.updated_by,
  updated_at = NOW()
RETURNING course_id, allowed_axes, default_strategy, holdout_percent, monthly_token_budget,
          require_instructor_approval, student_optout_allowed, updated_by, updated_at,
          generation_paused, max_prewarm_variants, budget_period_start, tokens_used_period
`, courseID, s.AllowedAxes, s.DefaultStrategy, s.HoldoutPercent, s.MonthlyTokenBudget,
		s.RequireInstructorApproval, s.StudentOptoutAllowed, actorUserID,
		s.GenerationPaused, maxPrewarm,
	).Scan(
		&r.CourseID, &r.AllowedAxes, &r.DefaultStrategy, &r.HoldoutPercent, &r.MonthlyTokenBudget,
		&r.RequireInstructorApproval, &r.StudentOptoutAllowed, &updatedBy, &r.UpdatedAt,
		&r.GenerationPaused, &r.MaxPrewarmVariants, &periodStart, &r.TokensUsedPeriod,
	)
	if err != nil {
		return nil, err
	}
	r.UpdatedBy = updatedBy
	r.BudgetPeriodStart = periodStart
	if r.AllowedAxes == nil {
		r.AllowedAxes = []string{}
	}
	return &r, nil
}

// PatchPipelineSettings updates AC.4-only fields without touching axes/budget policy.
func PatchPipelineSettings(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	generationPaused *bool,
	maxPrewarm *int16,
	actorUserID uuid.UUID,
) (*SettingsRow, error) {
	// Ensure a row exists with defaults first.
	existing, err := GetSettings(ctx, pool, courseID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		def := DefaultSettings(courseID)
		if generationPaused != nil {
			def.GenerationPaused = *generationPaused
		}
		if maxPrewarm != nil && *maxPrewarm >= 0 {
			def.MaxPrewarmVariants = *maxPrewarm
		}
		return UpsertSettings(ctx, pool, courseID, def, actorUserID)
	}
	next := *existing
	if generationPaused != nil {
		next.GenerationPaused = *generationPaused
	}
	if maxPrewarm != nil && *maxPrewarm >= 0 {
		next.MaxPrewarmVariants = *maxPrewarm
	}
	return UpsertSettings(ctx, pool, courseID, next, actorUserID)
}

// ListUnits returns all units for a course ordered by created_at.
func ListUnits(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]UnitRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, course_id, target_kind, target_module_item_id, target_outcome_id,
       base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
       allowed_axes, status, created_by, created_at, updated_at,
       trigger_mode, mastery_freshness_days, content_version, min_fidelity,
       quarantined, quarantined_reason
FROM course.adaptive_content_units
WHERE course_id = $1
ORDER BY created_at ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UnitRow
	for rows.Next() {
		u, err := scanUnit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// GetUnit returns a unit by id within a course, or nil.
func GetUnit(ctx context.Context, pool *pgxpool.Pool, courseID, unitID uuid.UUID) (*UnitRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, course_id, target_kind, target_module_item_id, target_outcome_id,
       base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
       allowed_axes, status, created_by, created_at, updated_at,
       trigger_mode, mastery_freshness_days, content_version, min_fidelity,
       quarantined, quarantined_reason
FROM course.adaptive_content_units
WHERE course_id = $1 AND id = $2
`, courseID, unitID)
	u, err := scanUnit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// InsertUnit creates a new unit.
func InsertUnit(ctx context.Context, pool *pgxpool.Pool, u UnitRow) (*UnitRow, error) {
	if u.AllowedAxes == nil {
		u.AllowedAxes = []string{}
	}
	if u.Status == "" {
		u.Status = "draft"
	}
	if u.TriggerMode == "" {
		u.TriggerMode = "pre_quiz"
	}
	if u.ContentVersion <= 0 {
		u.ContentVersion = 1
	}
	if u.MinFidelity <= 0 {
		u.MinFidelity = 0.85
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.adaptive_content_units (
  course_id, target_kind, target_module_item_id, target_outcome_id,
  base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
  allowed_axes, status, created_by, trigger_mode, mastery_freshness_days,
  content_version, min_fidelity
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
RETURNING id, course_id, target_kind, target_module_item_id, target_outcome_id,
          base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
          allowed_axes, status, created_by, created_at, updated_at,
          trigger_mode, mastery_freshness_days, content_version, min_fidelity,
          quarantined, quarantined_reason
`, u.CourseID, u.TargetKind, u.TargetModuleItemID, u.TargetOutcomeID,
		u.BaseContentItemID, u.PreAssessmentItemID, u.PostAssessmentItemID,
		u.AllowedAxes, u.Status, u.CreatedBy, u.TriggerMode, u.MasteryFreshnessDays,
		u.ContentVersion, u.MinFidelity,
	)
	out, err := scanUnit(row)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateUnit replaces mutable fields on an existing unit.
func UpdateUnit(ctx context.Context, pool *pgxpool.Pool, u UnitRow) (*UnitRow, error) {
	if u.AllowedAxes == nil {
		u.AllowedAxes = []string{}
	}
	if u.TriggerMode == "" {
		u.TriggerMode = "pre_quiz"
	}
	if u.MinFidelity <= 0 {
		u.MinFidelity = 0.85
	}
	if u.ContentVersion <= 0 {
		u.ContentVersion = 1
	}
	row := pool.QueryRow(ctx, `
UPDATE course.adaptive_content_units
SET target_kind = $3,
    target_module_item_id = $4,
    target_outcome_id = $5,
    base_content_item_id = $6,
    pre_assessment_item_id = $7,
    post_assessment_item_id = $8,
    allowed_axes = $9,
    status = $10,
    trigger_mode = $11,
    mastery_freshness_days = $12,
    min_fidelity = $13,
    updated_at = NOW()
WHERE course_id = $1 AND id = $2
RETURNING id, course_id, target_kind, target_module_item_id, target_outcome_id,
          base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
          allowed_axes, status, created_by, created_at, updated_at,
          trigger_mode, mastery_freshness_days, content_version, min_fidelity,
          quarantined, quarantined_reason
`, u.CourseID, u.ID, u.TargetKind, u.TargetModuleItemID, u.TargetOutcomeID,
		u.BaseContentItemID, u.PreAssessmentItemID, u.PostAssessmentItemID,
		u.AllowedAxes, u.Status, u.TriggerMode, u.MasteryFreshnessDays, u.MinFidelity,
	)
	out, err := scanUnit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteUnit deletes a unit; returns true if a row was deleted.
func DeleteUnit(ctx context.Context, pool *pgxpool.Pool, courseID, unitID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM course.adaptive_content_units
WHERE course_id = $1 AND id = $2
`, courseID, unitID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// StructureItemBelongsToCourse reports whether itemID is a structure item of courseID.
func StructureItemBelongsToCourse(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM course.course_structure_items
  WHERE id = $1 AND course_id = $2
)`, itemID, courseID).Scan(&ok)
	return ok, err
}

// OutcomeBelongsToCourse reports whether outcomeID is a learning outcome of courseID.
func OutcomeBelongsToCourse(ctx context.Context, pool *pgxpool.Pool, courseID, outcomeID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM course.course_learning_outcomes
  WHERE id = $1 AND course_id = $2
)`, outcomeID, courseID).Scan(&ok)
	return ok, err
}

// InsertEvent appends an adaptive_content_events audit row.
func InsertEvent(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	unitID *uuid.UUID,
	actorUserID *uuid.UUID,
	subjectUserID *uuid.UUID,
	eventType string,
	detail any,
) error {
	var detailJSON []byte
	if detail == nil {
		detailJSON = []byte("{}")
	} else {
		b, err := json.Marshal(detail)
		if err != nil {
			return err
		}
		detailJSON = b
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.adaptive_content_events (
  course_id, unit_id, actor_user_id, subject_user_id, event_type, detail_json
) VALUES ($1, $2, $3, $4, $5, $6::jsonb)
`, courseID, unitID, actorUserID, subjectUserID, eventType, string(detailJSON))
	return err
}

// CountEnabledCourses returns how many courses have adaptive_content_enabled = true.
func CountEnabledCourses(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.courses WHERE adaptive_content_enabled = TRUE
`).Scan(&n)
	return n, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanUnit(row scannable) (UnitRow, error) {
	var u UnitRow
	err := row.Scan(
		&u.ID, &u.CourseID, &u.TargetKind, &u.TargetModuleItemID, &u.TargetOutcomeID,
		&u.BaseContentItemID, &u.PreAssessmentItemID, &u.PostAssessmentItemID,
		&u.AllowedAxes, &u.Status, &u.CreatedBy, &u.CreatedAt, &u.UpdatedAt,
		&u.TriggerMode, &u.MasteryFreshnessDays, &u.ContentVersion, &u.MinFidelity,
		&u.Quarantined, &u.QuarantinedReason,
	)
	if err != nil {
		return UnitRow{}, err
	}
	if u.AllowedAxes == nil {
		u.AllowedAxes = []string{}
	}
	if u.TriggerMode == "" {
		u.TriggerMode = "pre_quiz"
	}
	if u.ContentVersion <= 0 {
		u.ContentVersion = 1
	}
	if u.MinFidelity <= 0 {
		u.MinFidelity = 0.85
	}
	return u, nil
}
