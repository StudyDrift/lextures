package adaptivecontent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutcomeRow is a course.adaptation_outcomes row (AC.7).
type OutcomeRow struct {
	ServingID      uuid.UUID
	PreScorePct    *float32
	PostScorePct   *float32
	MasteryBefore  *float32
	MasteryAfter   *float32
	Lift           *float32
	EmphasisMode   *string
	WasHoldout     bool
	PostAttemptID  *uuid.UUID
	MeasuredAt     time.Time
	VariantID      *uuid.UUID // joined from servings when loaded for aggregates
}

// OutcomeUpsert is the input for recording a post-assessment outcome.
type OutcomeUpsert struct {
	ServingID     uuid.UUID
	PreScorePct   *float32
	PostScorePct  *float32
	MasteryBefore *float32
	MasteryAfter  *float32
	Lift          *float32
	EmphasisMode  *string
	WasHoldout    bool
	PostAttemptID *uuid.UUID
}

// EffectivenessCacheRow is analytics.adaptive_content_effectiveness.
type EffectivenessCacheRow struct {
	UnitID                     uuid.UUID
	CourseID                   uuid.UUID
	NTreatment                 int
	NHoldout                   int
	MeanLiftTreatment          *float32
	MeanLiftHoldout            *float32
	TreatmentMinusHoldout      *float32
	DiffStdError               *float32
	MeanMasteryDeltaTreatment  *float32
	MeanMasteryDeltaHoldout    *float32
	Verdict                    string
	RefreshedAt                time.Time
}

// ModeEffectivenessRow is analytics.adaptive_content_effectiveness_by_mode.
type ModeEffectivenessRow struct {
	UnitID       uuid.UUID
	EmphasisMode string
	N            int
	MeanLift     *float32
}

// VariantEffectivenessRow is analytics.adaptive_content_effectiveness_by_variant.
type VariantEffectivenessRow struct {
	UnitID    uuid.UUID
	VariantID *uuid.UUID
	N         int
	MeanLift  *float32
}

// OutcomeLiftSample is one lift observation used for aggregation.
type OutcomeLiftSample struct {
	Lift          float32
	MasteryDelta  *float32
	WasHoldout    bool
	EmphasisMode  string
	VariantID     *uuid.UUID
}

// UpsertOutcome inserts or replaces the outcome for a serving (idempotent on serving_id).
func UpsertOutcome(ctx context.Context, pool *pgxpool.Pool, in OutcomeUpsert) (*OutcomeRow, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO course.adaptation_outcomes (
  serving_id, pre_score_pct, post_score_pct, mastery_before, mastery_after, lift,
  emphasis_mode, was_holdout, post_attempt_id, measured_at
) VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, NOW()
)
ON CONFLICT (serving_id) DO UPDATE SET
  pre_score_pct = EXCLUDED.pre_score_pct,
  post_score_pct = EXCLUDED.post_score_pct,
  mastery_before = EXCLUDED.mastery_before,
  mastery_after = EXCLUDED.mastery_after,
  lift = EXCLUDED.lift,
  emphasis_mode = EXCLUDED.emphasis_mode,
  was_holdout = EXCLUDED.was_holdout,
  post_attempt_id = EXCLUDED.post_attempt_id,
  measured_at = NOW()
RETURNING serving_id, pre_score_pct, post_score_pct, mastery_before, mastery_after, lift,
          emphasis_mode, was_holdout, post_attempt_id, measured_at
`, in.ServingID, in.PreScorePct, in.PostScorePct, in.MasteryBefore, in.MasteryAfter, in.Lift,
		in.EmphasisMode, in.WasHoldout, in.PostAttemptID)
	return scanOutcome(row)
}

// GetOutcome returns the outcome for a serving, or nil.
func GetOutcome(ctx context.Context, pool *pgxpool.Pool, servingID uuid.UUID) (*OutcomeRow, error) {
	row := pool.QueryRow(ctx, `
SELECT serving_id, pre_score_pct, post_score_pct, mastery_before, mastery_after, lift,
       emphasis_mode, was_holdout, post_attempt_id, measured_at
FROM course.adaptation_outcomes
WHERE serving_id = $1
`, servingID)
	o, err := scanOutcome(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return o, err
}

// ListActiveUnitsByPostAssessment returns active units whose post_assessment_item_id matches.
func ListActiveUnitsByPostAssessment(ctx context.Context, pool *pgxpool.Pool, courseID, postAssessmentItemID uuid.UUID) ([]UnitRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, course_id, target_kind, target_module_item_id, target_outcome_id,
       base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
       allowed_axes, status, created_by, created_at, updated_at,
       trigger_mode, mastery_freshness_days, content_version, min_fidelity
FROM course.adaptive_content_units
WHERE course_id = $1
  AND post_assessment_item_id = $2
  AND status = 'active'
ORDER BY created_at ASC
`, courseID, postAssessmentItemID)
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

// GetLatestServingForEnrollment returns the most recent serving for a unit+enrollment, or nil.
func GetLatestServingForEnrollment(ctx context.Context, pool *pgxpool.Pool, unitID, enrollmentID uuid.UUID) (*ServingRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, unit_id, enrollment_id, profile_id, variant_id, was_holdout, was_fallback,
       served_at, content_version, view_count, first_viewed_at, view_original_clicks
FROM course.adaptation_servings
WHERE unit_id = $1 AND enrollment_id = $2
ORDER BY served_at DESC, content_version DESC
LIMIT 1
`, unitID, enrollmentID)
	s, err := scanServing(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// ListOutcomeLiftSamples returns lift samples for a unit (only rows with non-null lift).
func ListOutcomeLiftSamples(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) ([]OutcomeLiftSample, error) {
	rows, err := pool.Query(ctx, `
SELECT o.lift,
       CASE WHEN o.mastery_before IS NOT NULL AND o.mastery_after IS NOT NULL
            THEN o.mastery_after - o.mastery_before END AS mastery_delta,
       o.was_holdout,
       COALESCE(o.emphasis_mode, 'unknown') AS emphasis_mode,
       s.variant_id
FROM course.adaptation_outcomes o
INNER JOIN course.adaptation_servings s ON s.id = o.serving_id
WHERE s.unit_id = $1
  AND o.lift IS NOT NULL
`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutcomeLiftSample
	for rows.Next() {
		var s OutcomeLiftSample
		if err := rows.Scan(&s.Lift, &s.MasteryDelta, &s.WasHoldout, &s.EmphasisMode, &s.VariantID); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ListUnitsWithPostAssessment returns active units in a course that have a post-assessment bound.
func ListUnitsWithPostAssessment(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]UnitRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, course_id, target_kind, target_module_item_id, target_outcome_id,
       base_content_item_id, pre_assessment_item_id, post_assessment_item_id,
       allowed_axes, status, created_by, created_at, updated_at,
       trigger_mode, mastery_freshness_days, content_version, min_fidelity
FROM course.adaptive_content_units
WHERE course_id = $1
  AND post_assessment_item_id IS NOT NULL
  AND status IN ('active', 'paused')
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

// ListCourseIDsWithAdaptiveContentEnabled returns course ids with ACE on.
func ListCourseIDsWithAdaptiveContentEnabled(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT id FROM course.courses WHERE adaptive_content_enabled = TRUE
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// UpsertEffectivenessCache writes the per-unit effectiveness aggregate row.
func UpsertEffectivenessCache(ctx context.Context, pool *pgxpool.Pool, row EffectivenessCacheRow) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.adaptive_content_effectiveness (
  unit_id, course_id, n_treatment, n_holdout,
  mean_lift_treatment, mean_lift_holdout, treatment_minus_holdout, diff_std_error,
  mean_mastery_delta_treatment, mean_mastery_delta_holdout, verdict, refreshed_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW()
)
ON CONFLICT (unit_id) DO UPDATE SET
  course_id = EXCLUDED.course_id,
  n_treatment = EXCLUDED.n_treatment,
  n_holdout = EXCLUDED.n_holdout,
  mean_lift_treatment = EXCLUDED.mean_lift_treatment,
  mean_lift_holdout = EXCLUDED.mean_lift_holdout,
  treatment_minus_holdout = EXCLUDED.treatment_minus_holdout,
  diff_std_error = EXCLUDED.diff_std_error,
  mean_mastery_delta_treatment = EXCLUDED.mean_mastery_delta_treatment,
  mean_mastery_delta_holdout = EXCLUDED.mean_mastery_delta_holdout,
  verdict = EXCLUDED.verdict,
  refreshed_at = NOW()
`, row.UnitID, row.CourseID, row.NTreatment, row.NHoldout,
		row.MeanLiftTreatment, row.MeanLiftHoldout, row.TreatmentMinusHoldout, row.DiffStdError,
		row.MeanMasteryDeltaTreatment, row.MeanMasteryDeltaHoldout, row.Verdict)
	return err
}

// ReplaceModeEffectiveness replaces per-mode rows for a unit.
func ReplaceModeEffectiveness(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, rows []ModeEffectivenessRow) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM analytics.adaptive_content_effectiveness_by_mode WHERE unit_id = $1`, unitID); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := tx.Exec(ctx, `
INSERT INTO analytics.adaptive_content_effectiveness_by_mode (unit_id, emphasis_mode, n, mean_lift)
VALUES ($1, $2, $3, $4)
`, unitID, r.EmphasisMode, r.N, r.MeanLift); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ReplaceVariantEffectiveness replaces per-variant rows for a unit.
func ReplaceVariantEffectiveness(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID, rows []VariantEffectivenessRow) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM analytics.adaptive_content_effectiveness_by_variant WHERE unit_id = $1`, unitID); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := tx.Exec(ctx, `
INSERT INTO analytics.adaptive_content_effectiveness_by_variant (unit_id, variant_id, n, mean_lift)
VALUES ($1, $2, $3, $4)
`, unitID, r.VariantID, r.N, r.MeanLift); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// GetEffectivenessCache returns the cached unit effectiveness, or nil.
func GetEffectivenessCache(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) (*EffectivenessCacheRow, error) {
	row := pool.QueryRow(ctx, `
SELECT unit_id, course_id, n_treatment, n_holdout,
       mean_lift_treatment, mean_lift_holdout, treatment_minus_holdout, diff_std_error,
       mean_mastery_delta_treatment, mean_mastery_delta_holdout, verdict, refreshed_at
FROM analytics.adaptive_content_effectiveness
WHERE unit_id = $1
`, unitID)
	var r EffectivenessCacheRow
	err := row.Scan(
		&r.UnitID, &r.CourseID, &r.NTreatment, &r.NHoldout,
		&r.MeanLiftTreatment, &r.MeanLiftHoldout, &r.TreatmentMinusHoldout, &r.DiffStdError,
		&r.MeanMasteryDeltaTreatment, &r.MeanMasteryDeltaHoldout, &r.Verdict, &r.RefreshedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListEffectivenessForCourse returns cached effectiveness for all units in a course.
func ListEffectivenessForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]EffectivenessCacheRow, error) {
	rows, err := pool.Query(ctx, `
SELECT unit_id, course_id, n_treatment, n_holdout,
       mean_lift_treatment, mean_lift_holdout, treatment_minus_holdout, diff_std_error,
       mean_mastery_delta_treatment, mean_mastery_delta_holdout, verdict, refreshed_at
FROM analytics.adaptive_content_effectiveness
WHERE course_id = $1
ORDER BY refreshed_at DESC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EffectivenessCacheRow
	for rows.Next() {
		var r EffectivenessCacheRow
		if err := rows.Scan(
			&r.UnitID, &r.CourseID, &r.NTreatment, &r.NHoldout,
			&r.MeanLiftTreatment, &r.MeanLiftHoldout, &r.TreatmentMinusHoldout, &r.DiffStdError,
			&r.MeanMasteryDeltaTreatment, &r.MeanMasteryDeltaHoldout, &r.Verdict, &r.RefreshedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListModeEffectiveness returns per-mode rows for a unit.
func ListModeEffectiveness(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) ([]ModeEffectivenessRow, error) {
	rows, err := pool.Query(ctx, `
SELECT unit_id, emphasis_mode, n, mean_lift
FROM analytics.adaptive_content_effectiveness_by_mode
WHERE unit_id = $1
ORDER BY emphasis_mode ASC
`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ModeEffectivenessRow
	for rows.Next() {
		var r ModeEffectivenessRow
		if err := rows.Scan(&r.UnitID, &r.EmphasisMode, &r.N, &r.MeanLift); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListVariantEffectiveness returns per-variant rows for a unit.
func ListVariantEffectiveness(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) ([]VariantEffectivenessRow, error) {
	rows, err := pool.Query(ctx, `
SELECT unit_id, variant_id, n, mean_lift
FROM analytics.adaptive_content_effectiveness_by_variant
WHERE unit_id = $1
ORDER BY n DESC
`, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VariantEffectivenessRow
	for rows.Next() {
		var r VariantEffectivenessRow
		if err := rows.Scan(&r.UnitID, &r.VariantID, &r.N, &r.MeanLift); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetPreviousVerdict returns the prior cached verdict for a unit (for regressing-alert edge detect).
func GetPreviousVerdict(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) (string, error) {
	var v string
	err := pool.QueryRow(ctx, `
SELECT verdict FROM analytics.adaptive_content_effectiveness WHERE unit_id = $1
`, unitID).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// QuizAttemptScorePct returns score_percent for a submitted attempt, or nil.
func QuizAttemptScorePct(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID) (*float32, error) {
	var pct *float32
	err := pool.QueryRow(ctx, `
SELECT score_percent FROM course.quiz_attempts
WHERE id = $1 AND status = 'submitted'
`, attemptID).Scan(&pct)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return pct, err
}

// AdaptivePostScoreForEnrollment returns the latest post_score_pct and holdout flag for a
// student enrollment on an outcome-targeted adaptive unit (FR-7 outcomes-report contribution).
func AdaptivePostScoreForEnrollment(
	ctx context.Context,
	q interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	outcomeID, enrollmentID uuid.UUID,
) (score *float32, wasHoldout bool, ok bool, err error) {
	var pct *float32
	var holdout bool
	scanErr := q.QueryRow(ctx, `
SELECT o.post_score_pct, o.was_holdout
FROM course.adaptive_content_units u
INNER JOIN course.adaptation_servings s ON s.unit_id = u.id AND s.enrollment_id = $2
INNER JOIN course.adaptation_outcomes o ON o.serving_id = s.id
WHERE u.target_kind = 'outcome'
  AND u.target_outcome_id = $1
  AND u.post_assessment_item_id IS NOT NULL
  AND o.post_score_pct IS NOT NULL
ORDER BY o.measured_at DESC
LIMIT 1
`, outcomeID, enrollmentID).Scan(&pct, &holdout)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return nil, false, false, nil
	}
	if scanErr != nil {
		return nil, false, false, scanErr
	}
	return pct, holdout, true, nil
}

// CourseCodeForID returns the course_code for a course id.
func CourseCodeForID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (string, error) {
	var code string
	err := pool.QueryRow(ctx, `SELECT course_code FROM course.courses WHERE id = $1`, courseID).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return code, err
}

func scanOutcome(row scannable) (*OutcomeRow, error) {
	var o OutcomeRow
	err := row.Scan(
		&o.ServingID, &o.PreScorePct, &o.PostScorePct, &o.MasteryBefore, &o.MasteryAfter, &o.Lift,
		&o.EmphasisMode, &o.WasHoldout, &o.PostAttemptID, &o.MeasuredAt,
	)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
