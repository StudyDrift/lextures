package adaptivecontent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContestRow is course.adaptive_content_contests.
type ContestRow struct {
	ID            uuid.UUID
	CourseID      uuid.UUID
	UnitID        uuid.UUID
	ServingID     *uuid.UUID
	StudentUserID uuid.UUID
	Reason        *string
	Status        string
	ResolvedBy    *uuid.UUID
	CreatedAt     time.Time
	ResolvedAt    *time.Time
}

// FairnessRow is analytics.adaptive_content_fairness.
type FairnessRow struct {
	ID            uuid.UUID
	CourseID      uuid.UUID
	Dimension     string
	GroupLabel    string
	N             int
	MeanFidelity  *float32
	CoveragePct   *float32
	MeanLift      *float32
	DisparityFlag bool
	ComputedAt    time.Time
}

// FairnessUpsert is the write shape for ReplaceFairnessRows.
type FairnessUpsert struct {
	CourseID      uuid.UUID
	Dimension     string
	GroupLabel    string
	N             int
	MeanFidelity  *float32
	CoveragePct   *float32
	MeanLift      *float32
	DisparityFlag bool
}

// FairnessRawCell is an uns-suppressed aggregate collected for fairness audit.
type FairnessRawCell struct {
	Dimension    string
	GroupLabel   string
	N            int
	MeanFidelity *float64
	CoveragePct  *float64
	MeanLift     *float64
}

// GetOrgAdaptiveContentEnabled returns the org toggle: nil = no opinion, true/false otherwise.
func GetOrgAdaptiveContentEnabled(ctx context.Context, pool *pgxpool.Pool) (*bool, error) {
	if pool == nil {
		return nil, nil
	}
	var v *bool
	err := pool.QueryRow(ctx, `
SELECT adaptive_content_org_enabled
FROM settings.platform_app_settings
WHERE id = 1
`).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

// SetOrgAdaptiveContentEnabled sets the org ACE toggle (nil clears to no opinion).
func SetOrgAdaptiveContentEnabled(ctx context.Context, pool *pgxpool.Pool, enabled *bool) error {
	if pool == nil {
		return errors.New("adaptivecontent: nil pool")
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO settings.platform_app_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING
`); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
UPDATE settings.platform_app_settings
SET adaptive_content_org_enabled = $1, updated_at = NOW()
WHERE id = 1
`, enabled)
	return err
}

// GetDurableKillSwitch reads the durable admin kill-switch.
func GetDurableKillSwitch(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	if pool == nil {
		return false, nil
	}
	var engaged bool
	err := pool.QueryRow(ctx, `
SELECT COALESCE(adaptive_content_kill_switch, FALSE)
FROM settings.platform_app_settings
WHERE id = 1
`).Scan(&engaged)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return engaged, err
}

// SetDurableKillSwitch sets the durable admin kill-switch.
func SetDurableKillSwitch(ctx context.Context, pool *pgxpool.Pool, engaged bool) error {
	if pool == nil {
		return errors.New("adaptivecontent: nil pool")
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO settings.platform_app_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING
`); err != nil {
		return err
	}
	_, err := pool.Exec(ctx, `
UPDATE settings.platform_app_settings
SET adaptive_content_kill_switch = $1, updated_at = NOW()
WHERE id = 1
`, engaged)
	return err
}

// IsCourseQuarantined reports course-wide ACE quarantine.
func IsCourseQuarantined(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (bool, error) {
	var q bool
	err := pool.QueryRow(ctx, `
SELECT COALESCE(adaptive_content_quarantined, FALSE)
FROM course.courses WHERE id = $1
`, courseID).Scan(&q)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return q, err
}

// SetCourseQuarantine sets or clears course-wide ACE quarantine.
func SetCourseQuarantine(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, quarantined bool, reason string) error {
	var reasonPtr *string
	if quarantined && reason != "" {
		reasonPtr = &reason
	}
	_, err := pool.Exec(ctx, `
UPDATE course.courses
SET adaptive_content_quarantined = $2,
    adaptive_content_quarantined_reason = $3
WHERE id = $1
`, courseID, quarantined, reasonPtr)
	return err
}

// SetUnitQuarantine sets or clears unit quarantine. Returns false if unit not found.
func SetUnitQuarantine(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, unitID uuid.UUID,
	quarantined bool,
	reason string,
	actor uuid.UUID,
) (bool, error) {
	var reasonPtr *string
	var actorPtr *uuid.UUID
	var at *time.Time
	if quarantined {
		if reason != "" {
			reasonPtr = &reason
		}
		actorPtr = &actor
		now := time.Now().UTC()
		at = &now
	}
	tag, err := pool.Exec(ctx, `
UPDATE course.adaptive_content_units
SET quarantined = $3,
    quarantined_reason = $4,
    quarantined_at = $5,
    quarantined_by = $6,
    updated_at = NOW()
WHERE course_id = $1 AND id = $2
`, courseID, unitID, quarantined, reasonPtr, at, actorPtr)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// InsertContest creates a student contest record.
func InsertContest(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, unitID, studentUserID uuid.UUID,
	servingID *uuid.UUID,
	reason string,
) (*ContestRow, error) {
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.adaptive_content_contests (
  course_id, unit_id, serving_id, student_user_id, reason, status
) VALUES ($1, $2, $3, $4, $5, 'open')
RETURNING id, course_id, unit_id, serving_id, student_user_id, reason, status,
          resolved_by, created_at, resolved_at
`, courseID, unitID, servingID, studentUserID, reasonPtr)
	return scanContest(row)
}

// ListContestsForCourse returns contests newest first, optionally filtered by status.
func ListContestsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, status string, limit, offset int) ([]ContestRow, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = pool.Query(ctx, `
SELECT id, course_id, unit_id, serving_id, student_user_id, reason, status,
       resolved_by, created_at, resolved_at
FROM course.adaptive_content_contests
WHERE course_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4
`, courseID, status, limit, offset)
	} else {
		rows, err = pool.Query(ctx, `
SELECT id, course_id, unit_id, serving_id, student_user_id, reason, status,
       resolved_by, created_at, resolved_at
FROM course.adaptive_content_contests
WHERE course_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, courseID, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContestRow
	for rows.Next() {
		c, err := scanContest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	if out == nil {
		out = []ContestRow{}
	}
	return out, rows.Err()
}

// GetContest returns a contest scoped to a course, or nil.
func GetContest(ctx context.Context, pool *pgxpool.Pool, courseID, contestID uuid.UUID) (*ContestRow, error) {
	row := pool.QueryRow(ctx, `
SELECT id, course_id, unit_id, serving_id, student_user_id, reason, status,
       resolved_by, created_at, resolved_at
FROM course.adaptive_content_contests
WHERE course_id = $1 AND id = $2
`, courseID, contestID)
	c, err := scanContest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// ResolveContest updates contest status. Returns nil if not found.
func ResolveContest(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, contestID, resolver uuid.UUID,
	status string,
) (*ContestRow, error) {
	row := pool.QueryRow(ctx, `
UPDATE course.adaptive_content_contests
SET status = $3, resolved_by = $4, resolved_at = NOW()
WHERE course_id = $1 AND id = $2 AND status = 'open'
RETURNING id, course_id, unit_id, serving_id, student_user_id, reason, status,
          resolved_by, created_at, resolved_at
`, courseID, contestID, status, resolver)
	c, err := scanContest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// CountOpenContestsForUnit returns open contest count for auto-pause threshold.
func CountOpenContestsForUnit(ctx context.Context, pool *pgxpool.Pool, unitID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM course.adaptive_content_contests
WHERE unit_id = $1 AND status = 'open'
`, unitID).Scan(&n)
	return n, err
}

// CountOpenContestsPlatform returns open contests across all courses (oversight).
func CountOpenContestsPlatform(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.adaptive_content_contests WHERE status = 'open'
`).Scan(&n)
	return n, err
}

// CountDisparityFlags returns fairness cells with disparity_flag for a course (0 course = all).
func CountDisparityFlags(ctx context.Context, pool *pgxpool.Pool, courseID *uuid.UUID) (int64, error) {
	var n int64
	var err error
	if courseID != nil {
		err = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM analytics.adaptive_content_fairness
WHERE course_id = $1 AND disparity_flag = TRUE
`, *courseID).Scan(&n)
	} else {
		err = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM analytics.adaptive_content_fairness WHERE disparity_flag = TRUE
`).Scan(&n)
	}
	return n, err
}

// CountQuarantinedUnits returns quarantined unit count (platform-wide).
func CountQuarantinedUnits(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.adaptive_content_units WHERE quarantined = TRUE
`).Scan(&n)
	return n, err
}

// ListFairnessForCourse returns fairness cells for a course.
func ListFairnessForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]FairnessRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, course_id, dimension, group_label, n, mean_fidelity, coverage_pct,
       mean_lift, disparity_flag, computed_at
FROM analytics.adaptive_content_fairness
WHERE course_id = $1
ORDER BY dimension, group_label
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FairnessRow
	for rows.Next() {
		var r FairnessRow
		if err := rows.Scan(
			&r.ID, &r.CourseID, &r.Dimension, &r.GroupLabel, &r.N,
			&r.MeanFidelity, &r.CoveragePct, &r.MeanLift, &r.DisparityFlag, &r.ComputedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []FairnessRow{}
	}
	return out, rows.Err()
}

// ReplaceFairnessRows deletes existing cells for a course and inserts the new set.
func ReplaceFairnessRows(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, rows []FairnessUpsert) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM analytics.adaptive_content_fairness WHERE course_id = $1`, courseID); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := tx.Exec(ctx, `
INSERT INTO analytics.adaptive_content_fairness (
  course_id, dimension, group_label, n, mean_fidelity, coverage_pct, mean_lift, disparity_flag, computed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
`, r.CourseID, r.Dimension, r.GroupLabel, r.N, r.MeanFidelity, r.CoveragePct, r.MeanLift, r.DisparityFlag); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// CollectFairnessRaw gathers language/section/accommodation aggregates for a course.
// Uses lawfully-held proxy attributes only; never sends demographics to the model.
func CollectFairnessRaw(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]FairnessRawCell, error) {
	var out []FairnessRawCell

	// Language group: user locale (BCP 47), fallback "unknown".
	langRows, err := pool.Query(ctx, `
WITH served AS (
  SELECT s.variant_id, s.was_holdout, s.was_fallback,
         COALESCE(NULLIF(TRIM(u.locale), ''), 'unknown') AS grp,
         v.fidelity_score, o.lift
  FROM course.adaptation_servings s
  JOIN course.adaptive_content_units u2 ON u2.id = s.unit_id
  JOIN course.course_enrollments e ON e.id = s.enrollment_id
  JOIN "user".users u ON u.id = e.user_id
  LEFT JOIN course.content_variants v ON v.id = s.variant_id
  LEFT JOIN course.adaptation_outcomes o ON o.serving_id = s.id
  WHERE u2.course_id = $1
)
SELECT grp,
       COUNT(*)::int AS n,
       AVG(fidelity_score) FILTER (WHERE fidelity_score IS NOT NULL) AS mean_fid,
       (COUNT(*) FILTER (WHERE variant_id IS NOT NULL AND NOT was_holdout AND NOT was_fallback)::float
          / NULLIF(COUNT(*), 0)) AS coverage,
       AVG(lift) FILTER (WHERE lift IS NOT NULL) AS mean_lift
FROM served
GROUP BY grp
`, courseID)
	if err != nil {
		return nil, err
	}
	defer langRows.Close()
	for langRows.Next() {
		cell, err := scanFairnessRaw(langRows, "language")
		if err != nil {
			return nil, err
		}
		out = append(out, cell)
	}
	if err := langRows.Err(); err != nil {
		return nil, err
	}

	// Section: course_sections via enrollment section_id when present.
	secRows, err := pool.Query(ctx, `
WITH served AS (
  SELECT s.variant_id, s.was_holdout, s.was_fallback,
         COALESCE(NULLIF(TRIM(sec.section_code), ''), 'unsectioned') AS grp,
         v.fidelity_score, o.lift
  FROM course.adaptation_servings s
  JOIN course.adaptive_content_units u2 ON u2.id = s.unit_id
  JOIN course.course_enrollments e ON e.id = s.enrollment_id
  LEFT JOIN course.course_sections sec ON sec.id = e.section_id
  LEFT JOIN course.content_variants v ON v.id = s.variant_id
  LEFT JOIN course.adaptation_outcomes o ON o.serving_id = s.id
  WHERE u2.course_id = $1
)
SELECT grp,
       COUNT(*)::int,
       AVG(fidelity_score) FILTER (WHERE fidelity_score IS NOT NULL),
       (COUNT(*) FILTER (WHERE variant_id IS NOT NULL AND NOT was_holdout AND NOT was_fallback)::float
          / NULLIF(COUNT(*), 0)),
       AVG(lift) FILTER (WHERE lift IS NOT NULL)
FROM served
GROUP BY grp
`, courseID)
	if err != nil {
		return nil, err
	}
	defer secRows.Close()
	for secRows.Next() {
		cell, err := scanFairnessRaw(secRows, "section")
		if err != nil {
			return nil, err
		}
		out = append(out, cell)
	}
	if err := secRows.Err(); err != nil {
		return nil, err
	}

	// Accommodation flag: active course.student_accommodations → "yes" else "no".
	accRows, err := pool.Query(ctx, `
WITH served AS (
  SELECT s.variant_id, s.was_holdout, s.was_fallback, e.user_id,
         v.fidelity_score, o.lift,
         CASE WHEN EXISTS (
           SELECT 1 FROM course.student_accommodations sa
           WHERE sa.user_id = e.user_id
             AND sa.active = TRUE
             AND (sa.course_id IS NULL OR sa.course_id = $1)
         ) THEN 'yes' ELSE 'no' END AS grp
  FROM course.adaptation_servings s
  JOIN course.adaptive_content_units u2 ON u2.id = s.unit_id
  JOIN course.course_enrollments e ON e.id = s.enrollment_id
  LEFT JOIN course.content_variants v ON v.id = s.variant_id
  LEFT JOIN course.adaptation_outcomes o ON o.serving_id = s.id
  WHERE u2.course_id = $1
)
SELECT grp,
       COUNT(*)::int,
       AVG(fidelity_score) FILTER (WHERE fidelity_score IS NOT NULL),
       (COUNT(*) FILTER (WHERE variant_id IS NOT NULL AND NOT was_holdout AND NOT was_fallback)::float
          / NULLIF(COUNT(*), 0)),
       AVG(lift) FILTER (WHERE lift IS NOT NULL)
FROM served
GROUP BY grp
`, courseID)
	if err != nil {
		return nil, err
	}
	defer accRows.Close()
	for accRows.Next() {
		cell, err := scanFairnessRaw(accRows, "accommodation")
		if err != nil {
			return nil, err
		}
		out = append(out, cell)
	}
	if err := accRows.Err(); err != nil {
		return nil, err
	}

	if out == nil {
		out = []FairnessRawCell{}
	}
	return out, nil
}

func scanFairnessRaw(row scannable, dimension string) (FairnessRawCell, error) {
	var c FairnessRawCell
	c.Dimension = dimension
	err := row.Scan(&c.GroupLabel, &c.N, &c.MeanFidelity, &c.CoveragePct, &c.MeanLift)
	return c, err
}

func scanContest(row scannable) (*ContestRow, error) {
	var c ContestRow
	err := row.Scan(
		&c.ID, &c.CourseID, &c.UnitID, &c.ServingID, &c.StudentUserID,
		&c.Reason, &c.Status, &c.ResolvedBy, &c.CreatedAt, &c.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// SumAdaptiveContentCostUSD returns recent ACE AI cost for oversight (last 30 days).
func SumAdaptiveContentCostUSD(ctx context.Context, pool *pgxpool.Pool) (float64, error) {
	var n float64
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(cost_usd), 0)::float8
FROM analytics.ai_usage_log
WHERE feature = $1 AND created_at >= NOW() - INTERVAL '30 days'
`, FeatureAdaptiveContent).Scan(&n)
	return n, err
}

// CountGateBlocksRecent counts gate_block events in the last 7 days.
func CountGateBlocksRecent(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM course.adaptive_content_events
WHERE event_type = 'gate_block' AND created_at >= NOW() - INTERVAL '7 days'
`).Scan(&n)
	return n, err
}

// CountRegressingUnits returns units currently verdict=regressing.
func CountRegressingUnits(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM analytics.adaptive_content_effectiveness WHERE verdict = 'regressing'
`).Scan(&n)
	return n, err
}
