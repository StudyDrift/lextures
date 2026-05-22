-- Plan 9.5 — Course-level outcomes reporting: cohort aggregation cache, improvement notes, link weights.

ALTER TABLE course.course_outcome_links
    ADD COLUMN IF NOT EXISTS weight REAL NOT NULL DEFAULT 1.0;

ALTER TABLE course.course_outcome_links
    ADD CONSTRAINT course_outcome_links_weight_positive CHECK (weight > 0);

COMMENT ON COLUMN course.course_outcome_links.weight IS
    'Relative weight when rolling up aligned assessment scores for accreditation reporting (plan 9.5).';

CREATE TABLE IF NOT EXISTS analytics.course_outcomes_report_config (
    course_id UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    mastery_threshold REAL NOT NULL DEFAULT 70.0
        CHECK (mastery_threshold > 0 AND mastery_threshold <= 100),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE analytics.course_outcomes_report_config IS
    'Per-course default mastery threshold (percent) for outcomes reporting (plan 9.5).';

CREATE TABLE IF NOT EXISTS analytics.outcome_improvement_notes (
    outcome_id UUID PRIMARY KEY REFERENCES course.course_learning_outcomes (id) ON DELETE CASCADE,
    note_text TEXT NOT NULL DEFAULT '' CHECK (char_length(note_text) <= 20000),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE analytics.outcome_improvement_notes IS
    'Instructor qualitative notes per learning outcome for accreditation portfolios (plan 9.5).';

CREATE TABLE IF NOT EXISTS analytics.outcomes_report_refresh (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO analytics.outcomes_report_refresh (id, refreshed_at) VALUES (1, NOW())
ON CONFLICT (id) DO NOTHING;

-- Per-student outcome scores; repopulated on refresh (plan 9.5).
CREATE TABLE IF NOT EXISTS analytics.outcomes_report_student (
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    outcome_id UUID NOT NULL REFERENCES course.course_learning_outcomes (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    section_id UUID REFERENCES course.course_sections (id) ON DELETE SET NULL,
    avg_score_pct REAL,
    assessed BOOLEAN NOT NULL DEFAULT FALSE,
    met BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (outcome_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_outcomes_report_student_course
    ON analytics.outcomes_report_student (course_id);
CREATE INDEX IF NOT EXISTS idx_outcomes_report_student_section
    ON analytics.outcomes_report_student (course_id, section_id);

-- Cohort-level aggregates (refreshed after student rows are rebuilt).
CREATE MATERIALIZED VIEW analytics.outcomes_report AS
SELECT
    lo.id AS outcome_id,
    lo.course_id,
    lo.title,
    lo.sort_order,
    COALESCE(cfg.mastery_threshold, 70.0)::real AS threshold,
    COUNT(DISTINCT e.user_id)::int AS n_students,
    COUNT(DISTINCT s.user_id) FILTER (WHERE s.assessed)::int AS n_assessed,
    AVG(s.avg_score_pct) FILTER (WHERE s.assessed)::real AS mean_score,
    COUNT(DISTINCT s.user_id) FILTER (WHERE s.met)::int AS n_met
FROM course.course_learning_outcomes lo
JOIN course.course_enrollments e
    ON e.course_id = lo.course_id AND e.active = TRUE AND e.role = 'student'
LEFT JOIN analytics.outcomes_report_student s
    ON s.outcome_id = lo.id AND s.user_id = e.user_id
LEFT JOIN analytics.course_outcomes_report_config cfg ON cfg.course_id = lo.course_id
GROUP BY lo.id, lo.course_id, lo.title, lo.sort_order, cfg.mastery_threshold;

CREATE UNIQUE INDEX IF NOT EXISTS idx_analytics_outcomes_report_outcome
    ON analytics.outcomes_report (outcome_id);

CREATE INDEX IF NOT EXISTS idx_analytics_outcomes_report_course
    ON analytics.outcomes_report (course_id);

COMMENT ON MATERIALIZED VIEW analytics.outcomes_report IS
    'Cached cohort achievement per learning outcome; refresh after student score rebuild (plan 9.5).';

REFRESH MATERIALIZED VIEW analytics.outcomes_report;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS outcomes_report_enabled BOOLEAN;
