package adaptivecontent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CourseReportRollupRow is analytics.adaptive_content_course_report.
type CourseReportRollupRow struct {
	CourseID           uuid.UUID
	NUnits             int
	NActiveUnits       int
	MeanLiftVsControl  *float32
	NHelping           int
	NRegressing        int
	NInsufficient      int
	NNoEffect          int
	TokensUsedPeriod   int64
	MonthlyTokenBudget int64
	RefreshedAt        time.Time
}

// CoverageRow is analytics.adaptive_content_coverage.
type CoverageRow struct {
	CourseID               uuid.UUID
	EligibleContentItems   int
	AdaptedUnits           int
	StudentsProfiled       int
	StudentsServedVariant  int
	StudentsHoldout        int
	RefreshedAt            time.Time
}

// UnitFidelityRow is mean fidelity for a unit's ready variants (AC.9 units-to-review).
type UnitFidelityRow struct {
	UnitID       uuid.UUID
	MeanFidelity *float32
	MinFidelity  float64
	NVariants    int
}

// AdminCourseRollupRow is one course row in the admin org report drill-down.
type AdminCourseRollupRow struct {
	CourseID              uuid.UUID
	CourseCode            string
	Title                 string
	NUnits                int
	NActiveUnits          int
	MeanLiftVsControl     *float32
	NRegressing           int
	NHelping              int
	TokensUsedPeriod      int64
	MonthlyTokenBudget    int64
	EligibleContentItems  int
	AdaptedUnits          int
	StudentsProfiled      int
	StudentsServedVariant int
	StudentsHoldout       int
	DisparityFlags        int64
	OpenContests          int64
	CoverageRefreshedAt   *time.Time
	ReportRefreshedAt     *time.Time
}

// RefreshCourseReportMaterializedView refreshes the AC.9 course report matview.
func RefreshCourseReportMaterializedView(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("adaptivecontent: nil pool")
	}
	_, err := pool.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.adaptive_content_course_report`)
	return err
}

// GetCourseReportRollup returns the cached course rollup, or nil when missing.
func GetCourseReportRollup(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*CourseReportRollupRow, error) {
	var r CourseReportRollupRow
	err := pool.QueryRow(ctx, `
SELECT course_id, n_units, n_active_units, mean_lift_vs_control,
       n_helping, n_regressing, n_insufficient, n_no_effect,
       tokens_used_period, monthly_token_budget, refreshed_at
FROM analytics.adaptive_content_course_report
WHERE course_id = $1
`, courseID).Scan(
		&r.CourseID, &r.NUnits, &r.NActiveUnits, &r.MeanLiftVsControl,
		&r.NHelping, &r.NRegressing, &r.NInsufficient, &r.NNoEffect,
		&r.TokensUsedPeriod, &r.MonthlyTokenBudget, &r.RefreshedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpsertCoverage writes a coverage snapshot for a course.
func UpsertCoverage(ctx context.Context, pool *pgxpool.Pool, row CoverageRow) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.adaptive_content_coverage (
  course_id, eligible_content_items, adapted_units,
  students_profiled, students_served_variant, students_holdout, refreshed_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (course_id) DO UPDATE SET
  eligible_content_items = EXCLUDED.eligible_content_items,
  adapted_units = EXCLUDED.adapted_units,
  students_profiled = EXCLUDED.students_profiled,
  students_served_variant = EXCLUDED.students_served_variant,
  students_holdout = EXCLUDED.students_holdout,
  refreshed_at = NOW()
`, row.CourseID, row.EligibleContentItems, row.AdaptedUnits,
		row.StudentsProfiled, row.StudentsServedVariant, row.StudentsHoldout)
	return err
}

// GetCoverage returns the coverage snapshot, or nil when missing.
func GetCoverage(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*CoverageRow, error) {
	var r CoverageRow
	err := pool.QueryRow(ctx, `
SELECT course_id, eligible_content_items, adapted_units,
       students_profiled, students_served_variant, students_holdout, refreshed_at
FROM analytics.adaptive_content_coverage
WHERE course_id = $1
`, courseID).Scan(
		&r.CourseID, &r.EligibleContentItems, &r.AdaptedUnits,
		&r.StudentsProfiled, &r.StudentsServedVariant, &r.StudentsHoldout, &r.RefreshedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ComputeCoverageSnapshot aggregates live coverage facts for a course (not served to clients directly).
func ComputeCoverageSnapshot(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (CoverageRow, error) {
	var r CoverageRow
	r.CourseID = courseID
	err := pool.QueryRow(ctx, `
SELECT
  (SELECT COUNT(DISTINCT base_content_item_id)::INTEGER
     FROM course.adaptive_content_units WHERE course_id = $1),
  (SELECT COUNT(*)::INTEGER
     FROM course.adaptive_content_units u
     WHERE u.course_id = $1
       AND EXISTS (
         SELECT 1 FROM course.content_variants v
         WHERE v.unit_id = u.id
           AND v.status IN ('approved', 'auto_served')
           AND v.content_version = u.content_version
       )),
  (SELECT COUNT(DISTINCT p.enrollment_id)::INTEGER
     FROM course.adaptation_profiles p
     INNER JOIN course.adaptive_content_units u ON u.id = p.unit_id
     WHERE u.course_id = $1),
  (SELECT COUNT(DISTINCT s.enrollment_id)::INTEGER
     FROM course.adaptation_servings s
     INNER JOIN course.adaptive_content_units u ON u.id = s.unit_id
     WHERE u.course_id = $1
       AND s.variant_id IS NOT NULL
       AND COALESCE(s.was_holdout, FALSE) = FALSE
       AND COALESCE(s.was_fallback, FALSE) = FALSE),
  (SELECT COUNT(DISTINCT s.enrollment_id)::INTEGER
     FROM course.adaptation_servings s
     INNER JOIN course.adaptive_content_units u ON u.id = s.unit_id
     WHERE u.course_id = $1 AND COALESCE(s.was_holdout, FALSE) = TRUE)
`, courseID).Scan(
		&r.EligibleContentItems, &r.AdaptedUnits,
		&r.StudentsProfiled, &r.StudentsServedVariant, &r.StudentsHoldout,
	)
	if err != nil {
		return r, err
	}
	r.RefreshedAt = time.Now().UTC()
	return r, nil
}

// RefreshCoverageForCourse recomputes and upserts coverage for one course.
func RefreshCoverageForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	snap, err := ComputeCoverageSnapshot(ctx, pool, courseID)
	if err != nil {
		return err
	}
	return UpsertCoverage(ctx, pool, snap)
}

// ListUnitMeanFidelity returns mean fidelity of ready variants per unit in a course.
func ListUnitMeanFidelity(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]UnitFidelityRow, error) {
	rows, err := pool.Query(ctx, `
SELECT u.id, u.min_fidelity,
       AVG(v.fidelity_score)::REAL AS mean_fidelity,
       COUNT(v.id)::INTEGER AS n_variants
FROM course.adaptive_content_units u
LEFT JOIN course.content_variants v
  ON v.unit_id = u.id
 AND v.status IN ('approved', 'auto_served', 'pending_review')
 AND v.content_version = u.content_version
WHERE u.course_id = $1
GROUP BY u.id, u.min_fidelity
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UnitFidelityRow
	for rows.Next() {
		var r UnitFidelityRow
		if err := rows.Scan(&r.UnitID, &r.MinFidelity, &r.MeanFidelity, &r.NVariants); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountEnabledACECourses returns courses with adaptive_content_enabled.
func CountEnabledACECourses(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.courses WHERE adaptive_content_enabled = TRUE
`).Scan(&n)
	return n, err
}

// SumCoverageStudentsImpacted returns sum of students_served_variant across coverage rows.
func SumCoverageStudentsImpacted(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(students_served_variant), 0)
FROM analytics.adaptive_content_coverage
`).Scan(&n)
	return n, err
}

// SumBudgetHeadroomTokens returns remaining tokens across courses with a finite monthly budget.
// Unlimited (budget=0) courses contribute 0 headroom.
func SumBudgetHeadroomTokens(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(
  CASE
    WHEN monthly_token_budget <= 0 THEN 0
    WHEN monthly_token_budget > tokens_used_period
      THEN monthly_token_budget - tokens_used_period
    ELSE 0
  END
), 0)
FROM course.adaptive_content_settings s
INNER JOIN course.courses c ON c.id = s.course_id
WHERE c.adaptive_content_enabled = TRUE
`).Scan(&n)
	return n, err
}

// MeanAggregateLiftVsControl averages treatment_minus_holdout across effectiveness cache.
func MeanAggregateLiftVsControl(ctx context.Context, pool *pgxpool.Pool) (*float32, error) {
	var v *float32
	err := pool.QueryRow(ctx, `
SELECT AVG(treatment_minus_holdout)::REAL
FROM analytics.adaptive_content_effectiveness
WHERE treatment_minus_holdout IS NOT NULL
`).Scan(&v)
	return v, err
}

// ListAdminCourseRollups returns per-course drill-down rows for the org report.
func ListAdminCourseRollups(ctx context.Context, pool *pgxpool.Pool, limit int) ([]AdminCourseRollupRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT c.id, c.course_code, c.title,
       COALESCE(r.n_units, 0), COALESCE(r.n_active_units, 0), r.mean_lift_vs_control,
       COALESCE(r.n_regressing, 0), COALESCE(r.n_helping, 0),
       COALESCE(r.tokens_used_period, s.tokens_used_period, 0),
       COALESCE(r.monthly_token_budget, s.monthly_token_budget, 0),
       COALESCE(cov.eligible_content_items, 0), COALESCE(cov.adapted_units, 0),
       COALESCE(cov.students_profiled, 0), COALESCE(cov.students_served_variant, 0),
       COALESCE(cov.students_holdout, 0),
       COALESCE(f.disparity_flags, 0),
       COALESCE(ct.open_contests, 0),
       cov.refreshed_at, r.refreshed_at
FROM course.courses c
LEFT JOIN analytics.adaptive_content_course_report r ON r.course_id = c.id
LEFT JOIN analytics.adaptive_content_coverage cov ON cov.course_id = c.id
LEFT JOIN course.adaptive_content_settings s ON s.course_id = c.id
LEFT JOIN LATERAL (
  SELECT COUNT(*)::BIGINT AS disparity_flags
  FROM analytics.adaptive_content_fairness af
  WHERE af.course_id = c.id AND af.disparity_flag = TRUE
) f ON TRUE
LEFT JOIN LATERAL (
  SELECT COUNT(*)::BIGINT AS open_contests
  FROM course.adaptive_content_contests ac
  WHERE ac.course_id = c.id AND ac.status = 'open'
) ct ON TRUE
WHERE c.adaptive_content_enabled = TRUE
ORDER BY COALESCE(r.n_regressing, 0) DESC, COALESCE(f.disparity_flags, 0) DESC, c.course_code ASC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminCourseRollupRow
	for rows.Next() {
		var r AdminCourseRollupRow
		if err := rows.Scan(
			&r.CourseID, &r.CourseCode, &r.Title,
			&r.NUnits, &r.NActiveUnits, &r.MeanLiftVsControl,
			&r.NRegressing, &r.NHelping,
			&r.TokensUsedPeriod, &r.MonthlyTokenBudget,
			&r.EligibleContentItems, &r.AdaptedUnits,
			&r.StudentsProfiled, &r.StudentsServedVariant, &r.StudentsHoldout,
			&r.DisparityFlags, &r.OpenContests,
			&r.CoverageRefreshedAt, &r.ReportRefreshedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
