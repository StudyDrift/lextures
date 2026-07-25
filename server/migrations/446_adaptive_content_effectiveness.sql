-- AC.7: Post-assessment outcomes extensions + effectiveness aggregate cache.

ALTER TABLE course.adaptation_outcomes
    ADD COLUMN IF NOT EXISTS emphasis_mode TEXT,
    ADD COLUMN IF NOT EXISTS was_holdout BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS post_attempt_id UUID REFERENCES course.quiz_attempts (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_ac_outcomes_measured
    ON course.adaptation_outcomes USING BRIN (measured_at);

COMMENT ON COLUMN course.adaptation_outcomes.emphasis_mode IS
    'AC.7: Denormalized emphasis_mode from the adaptation profile at measurement time.';
COMMENT ON COLUMN course.adaptation_outcomes.was_holdout IS
    'AC.7: Whether the serving was holdout/control (copied from adaptation_servings).';
COMMENT ON COLUMN course.adaptation_outcomes.post_attempt_id IS
    'AC.7: Quiz attempt that produced the post-assessment score.';

-- Per-unit effectiveness cache (refreshed by job; source of truth = per-serving rows).
CREATE TABLE IF NOT EXISTS analytics.adaptive_content_effectiveness (
    unit_id UUID PRIMARY KEY REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    n_treatment INTEGER NOT NULL DEFAULT 0,
    n_holdout INTEGER NOT NULL DEFAULT 0,
    mean_lift_treatment REAL,
    mean_lift_holdout REAL,
    treatment_minus_holdout REAL,
    diff_std_error REAL,
    mean_mastery_delta_treatment REAL,
    mean_mastery_delta_holdout REAL,
    verdict TEXT NOT NULL DEFAULT 'insufficient_data'
        CHECK (verdict IN ('helping', 'no_effect', 'insufficient_data', 'regressing')),
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ac_eff_course
    ON analytics.adaptive_content_effectiveness (course_id);

COMMENT ON TABLE analytics.adaptive_content_effectiveness IS
    'AC.7: Cached treatment-vs-holdout effectiveness per adaptive content unit.';

-- Per (unit, emphasis_mode) breakdown.
CREATE TABLE IF NOT EXISTS analytics.adaptive_content_effectiveness_by_mode (
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    emphasis_mode TEXT NOT NULL,
    n INTEGER NOT NULL DEFAULT 0,
    mean_lift REAL,
    PRIMARY KEY (unit_id, emphasis_mode)
);

COMMENT ON TABLE analytics.adaptive_content_effectiveness_by_mode IS
    'AC.7: Per-emphasis-mode mean lift for an adaptive content unit.';

-- Per (unit, variant) breakdown (variant_id NULL = base/holdout/fallback).
CREATE TABLE IF NOT EXISTS analytics.adaptive_content_effectiveness_by_variant (
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    variant_id UUID REFERENCES course.content_variants (id) ON DELETE CASCADE,
    n INTEGER NOT NULL DEFAULT 0,
    mean_lift REAL
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_ac_eff_by_variant
    ON analytics.adaptive_content_effectiveness_by_variant (
        unit_id,
        COALESCE(variant_id, '00000000-0000-0000-0000-000000000000'::uuid)
    );

CREATE INDEX IF NOT EXISTS idx_ac_eff_by_variant_unit
    ON analytics.adaptive_content_effectiveness_by_variant (unit_id);

COMMENT ON TABLE analytics.adaptive_content_effectiveness_by_variant IS
    'AC.7: Per-variant mean lift; NULL variant_id means base/control content.';

-- Tag outcomes-report student rows with adaptive arm for accreditation separation (FR-7).
ALTER TABLE analytics.outcomes_report_student
    ADD COLUMN IF NOT EXISTS adaptive_arm TEXT
        CHECK (adaptive_arm IS NULL OR adaptive_arm IN ('treatment', 'holdout'));

COMMENT ON COLUMN analytics.outcomes_report_student.adaptive_arm IS
    'AC.7: treatment|holdout when score includes adaptive-unit post-assessment; else null.';
