-- AC.6 — Student runtime serving records, view accounting, and per-course opt-out.

ALTER TABLE course.adaptation_servings
    ADD COLUMN IF NOT EXISTS content_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS view_count INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS first_viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS view_original_clicks INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN course.adaptation_servings.content_version IS
    'AC.6: adaptive_content_units.content_version at serve time; exposure key with enrollment.';
COMMENT ON COLUMN course.adaptation_servings.view_count IS
    'AC.6: Number of times this exposure was (re)opened; first insert = 1.';
COMMENT ON COLUMN course.adaptation_servings.first_viewed_at IS
    'AC.6: Timestamp of first serve for this (unit, enrollment, content_version).';
COMMENT ON COLUMN course.adaptation_servings.view_original_clicks IS
    'AC.6: Count of student "View original" toggles for this exposure.';

-- One serving row per exposure of a content version.
CREATE UNIQUE INDEX IF NOT EXISTS ux_ac_servings_exposure
    ON course.adaptation_servings (unit_id, enrollment_id, content_version);

-- Per-student, per-course opt-out of adaptive content.
CREATE TABLE IF NOT EXISTS course.adaptive_content_optouts (
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    opted_out BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (course_id, user_id)
);

COMMENT ON TABLE course.adaptive_content_optouts IS
    'AC.6: Per-course student opt-out of AI-adapted content. FERPA education record.';
