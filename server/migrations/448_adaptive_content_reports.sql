-- AC.9: Analytics, reporting & operability — course report rollups + coverage snapshot.

-- Course-level rollup (refreshed with AC.7 effectiveness / AC.9 report refresh).
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.adaptive_content_course_report AS
SELECT
    u.course_id,
    COUNT(*)::INTEGER                                            AS n_units,
    COUNT(*) FILTER (WHERE u.status = 'active')::INTEGER         AS n_active_units,
    AVG(e.treatment_minus_holdout)::REAL                         AS mean_lift_vs_control,
    COUNT(*) FILTER (WHERE e.verdict = 'helping')::INTEGER       AS n_helping,
    COUNT(*) FILTER (WHERE e.verdict = 'regressing')::INTEGER    AS n_regressing,
    COUNT(*) FILTER (WHERE e.verdict = 'insufficient_data')::INTEGER AS n_insufficient,
    COUNT(*) FILTER (WHERE e.verdict = 'no_effect')::INTEGER     AS n_no_effect,
    COALESCE(s.tokens_used_period, 0)::BIGINT                    AS tokens_used_period,
    COALESCE(s.monthly_token_budget, 0)::BIGINT                  AS monthly_token_budget,
    NOW()                                                        AS refreshed_at
FROM course.adaptive_content_units u
LEFT JOIN analytics.adaptive_content_effectiveness e ON e.unit_id = u.id
LEFT JOIN course.adaptive_content_settings s ON s.course_id = u.course_id
GROUP BY u.course_id, s.tokens_used_period, s.monthly_token_budget;

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_course_report
    ON analytics.adaptive_content_course_report (course_id);

COMMENT ON MATERIALIZED VIEW analytics.adaptive_content_course_report IS
    'AC.9: Cached course-level Adaptive Content report KPIs (effectiveness + budget).';

-- Coverage snapshot: eligible content items vs. adapted, students profiled/served (refreshed by job).
CREATE TABLE IF NOT EXISTS analytics.adaptive_content_coverage (
    course_id UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    eligible_content_items INTEGER NOT NULL DEFAULT 0,
    adapted_units INTEGER NOT NULL DEFAULT 0,
    students_profiled INTEGER NOT NULL DEFAULT 0,
    students_served_variant INTEGER NOT NULL DEFAULT 0,
    students_holdout INTEGER NOT NULL DEFAULT 0,
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE analytics.adaptive_content_coverage IS
    'AC.9: Per-course coverage snapshot for Adaptive Content instructor/admin reports.';
