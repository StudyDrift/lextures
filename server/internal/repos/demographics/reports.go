package demographics

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Title1Report is the aggregate demographic breakdown for a school org unit.
type Title1Report struct {
	SchoolID              uuid.UUID
	TotalStudents         int
	FreeLunchCount        int
	ReducedLunchCount     int
	EconomicDisadvantaged int
	EconomicPct           float64
	EllCount              int
	DisabilityCount       int
	HomelessCount         int
	MigrantCount          int
	RaceBreakdown         map[string]int
}

// SubgroupPerformance is one row in a disaggregated performance report.
type SubgroupPerformance struct {
	Label      string   `json:"label"`
	Count      int      `json:"count"`
	Suppressed bool     `json:"suppressed"`
	PassRate   *float64 `json:"passRate"`
}

// DisaggregatedReport holds performance metrics split by a demographic dimension.
type DisaggregatedReport struct {
	Dimension string                `json:"dimension"`
	Subgroups []SubgroupPerformance `json:"subgroups"`
}

// ApplySuppression returns whether a subgroup should be suppressed (n < 10) and the value to expose.
func ApplySuppression(count int, value float64) (suppressed bool, out *float64) {
	if count < MinSubgroupSize {
		return true, nil
	}
	v := math.Round(value*10) / 10
	return false, &v
}

// Title1AggregateReport builds the school-level Title I aggregate report.
func Title1AggregateReport(ctx context.Context, pool *pgxpool.Pool, schoolID uuid.UUID) (*Title1Report, error) {
	var total int
	err := pool.QueryRow(ctx, `
WITH RECURSIVE subtree AS (
    SELECT id FROM tenant.org_units WHERE id = $1
    UNION ALL
    SELECT ou.id FROM tenant.org_units ou
    INNER JOIN subtree s ON ou.parent_id = s.id
),
school_students AS (
    SELECT DISTINCT ce.user_id AS student_id
    FROM course.course_enrollments ce
    INNER JOIN course.courses c ON c.id = ce.course_id
    WHERE ce.role = 'student'
      AND c.org_unit_id IN (SELECT id FROM subtree)
)
SELECT COUNT(*)::int FROM school_students
`, schoolID).Scan(&total)
	if err != nil {
		return nil, err
	}

	report := &Title1Report{
		SchoolID:      schoolID,
		TotalStudents: total,
		RaceBreakdown: map[string]int{},
	}

	if total == 0 {
		return report, nil
	}

	err = pool.QueryRow(ctx, `
WITH RECURSIVE subtree AS (
    SELECT id FROM tenant.org_units WHERE id = $1
    UNION ALL
    SELECT ou.id FROM tenant.org_units ou
    INNER JOIN subtree s ON ou.parent_id = s.id
),
school_students AS (
    SELECT DISTINCT ce.user_id AS student_id
    FROM course.course_enrollments ce
    INNER JOIN course.courses c ON c.id = ce.course_id
    WHERE ce.role = 'student'
      AND c.org_unit_id IN (SELECT id FROM subtree)
)
SELECT
    COUNT(*) FILTER (WHERE sd.free_lunch IS TRUE)::int,
    COUNT(*) FILTER (WHERE sd.reduced_lunch IS TRUE)::int,
    COUNT(*) FILTER (WHERE sd.ell_status IS TRUE)::int,
    COUNT(*) FILTER (WHERE sd.disability_status IS TRUE)::int,
    COUNT(*) FILTER (WHERE sd.homeless_indicator IS TRUE)::int,
    COUNT(*) FILTER (WHERE sd.migrant_indicator IS TRUE)::int
FROM school_students ss
LEFT JOIN compliance.student_demographics sd ON sd.student_id = ss.student_id
`, schoolID).Scan(
		&report.FreeLunchCount, &report.ReducedLunchCount,
		&report.EllCount, &report.DisabilityCount,
		&report.HomelessCount, &report.MigrantCount,
	)
	if err != nil {
		return nil, err
	}

	report.EconomicDisadvantaged = report.FreeLunchCount + report.ReducedLunchCount
	if total > 0 {
		report.EconomicPct = math.Round((float64(report.EconomicDisadvantaged)/float64(total))*1000) / 10
	}

	raceRows, err := pool.Query(ctx, `
WITH RECURSIVE subtree AS (
    SELECT id FROM tenant.org_units WHERE id = $1
    UNION ALL
    SELECT ou.id FROM tenant.org_units ou
    INNER JOIN subtree s ON ou.parent_id = s.id
),
school_students AS (
    SELECT DISTINCT ce.user_id AS student_id
    FROM course.course_enrollments ce
    INNER JOIN course.courses c ON c.id = ce.course_id
    WHERE ce.role = 'student'
      AND c.org_unit_id IN (SELECT id FROM subtree)
)
SELECT COALESCE(sd.race_ethnicity_code, 'unknown'), COUNT(*)::int
FROM school_students ss
LEFT JOIN compliance.student_demographics sd ON sd.student_id = ss.student_id
GROUP BY COALESCE(sd.race_ethnicity_code, 'unknown')
`, schoolID)
	if err != nil {
		return nil, err
	}
	defer raceRows.Close()
	for raceRows.Next() {
		var code string
		var n int
		if err := raceRows.Scan(&code, &n); err != nil {
			return nil, err
		}
		report.RaceBreakdown[code] = n
	}
	return report, raceRows.Err()
}

// DisaggregatedPerformanceReport compares quiz pass rates across ELL subgroups.
func DisaggregatedPerformanceReport(ctx context.Context, pool *pgxpool.Pool, schoolID uuid.UUID, dimension string) (*DisaggregatedReport, error) {
	if dimension == "" {
		dimension = "ell"
	}
	report := &DisaggregatedReport{Dimension: dimension}

	switch dimension {
	case "ell":
		subgroups, err := ellPassRateSubgroups(ctx, pool, schoolID)
		if err != nil {
			return nil, err
		}
		report.Subgroups = subgroups
	default:
		report.Subgroups = []SubgroupPerformance{}
	}
	return report, nil
}

func ellPassRateSubgroups(ctx context.Context, pool *pgxpool.Pool, schoolID uuid.UUID) ([]SubgroupPerformance, error) {
	type row struct {
		label    string
		ell      bool
		count    int
		passed   int
		assessed int
	}
	rows, err := pool.Query(ctx, `
WITH RECURSIVE subtree AS (
    SELECT id FROM tenant.org_units WHERE id = $1
    UNION ALL
    SELECT ou.id FROM tenant.org_units ou
    INNER JOIN subtree s ON ou.parent_id = s.id
),
school_students AS (
    SELECT DISTINCT ce.user_id AS student_id
    FROM course.course_enrollments ce
    INNER JOIN course.courses c ON c.id = ce.course_id
    WHERE ce.role = 'student'
      AND c.org_unit_id IN (SELECT id FROM subtree)
),
quiz_scores AS (
    SELECT ss.student_id,
           COALESCE(sd.ell_status, false) AS ell_status,
           cg.points_earned,
           COALESCE(mq.points_worth, 0) AS points_worth
    FROM school_students ss
    LEFT JOIN compliance.student_demographics sd ON sd.student_id = ss.student_id
    INNER JOIN course.course_enrollments ce ON ce.user_id = ss.student_id AND ce.role = 'student'
    INNER JOIN course.courses c ON c.id = ce.course_id AND c.org_unit_id IN (SELECT id FROM subtree)
    INNER JOIN course.course_grades cg ON cg.student_user_id = ss.student_id AND cg.course_id = c.id
    INNER JOIN course.course_structure_items csi ON csi.id = cg.module_item_id AND csi.kind = 'quiz'
    LEFT JOIN course.module_quizzes mq ON mq.structure_item_id = csi.id
    WHERE COALESCE(mq.points_worth, 0) > 0
)
SELECT
    CASE WHEN ell_status THEN 'ELL' ELSE 'Non-ELL' END AS label,
    ell_status,
    COUNT(DISTINCT student_id)::int AS student_count,
    COUNT(*) FILTER (WHERE points_earned >= points_worth * 0.6)::int AS passed,
    COUNT(*)::int AS assessed
FROM quiz_scores
GROUP BY ell_status
`, schoolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SubgroupPerformance
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.label, &r.ell, &r.count, &r.passed, &r.assessed); err != nil {
			return nil, err
		}
		var passRate float64
		if r.assessed > 0 {
			passRate = (float64(r.passed) / float64(r.assessed)) * 100
		}
		suppressed, rate := ApplySuppression(r.count, passRate)
		out = append(out, SubgroupPerformance{
			Label:      r.label,
			Count:      r.count,
			Suppressed: suppressed,
			PassRate:   rate,
		})
	}
	return out, rows.Err()
}
