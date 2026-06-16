-- Plan 15.2 — Self-paced enrollment with no instructor.
-- Adds a self_paced course mode, open enrollment, optional module gating, and
-- per-enrollment learner item progress so learners can progress without an instructor.

CREATE TYPE course.course_mode AS ENUM ('instructor_led', 'self_paced');

ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS course_mode course.course_mode NOT NULL DEFAULT 'instructor_led';
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS open_enrollment BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS module_gating_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN course.courses.course_mode IS
    'instructor_led (default) or self_paced; self_paced runs without instructor involvement (plan 15.2).';
COMMENT ON COLUMN course.courses.open_enrollment IS
    'When true, any authenticated learner may self-enroll without instructor approval (plan 15.2).';
COMMENT ON COLUMN course.courses.module_gating_enabled IS
    'When true, a learner must complete module N before accessing module N+1 (plan 15.2).';

-- Per-enrollment, per-item learner progress for self-paced courses.
CREATE TABLE course.learner_item_progress (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id   UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    item_id         UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'not_started'
        CHECK (status IN ('not_started', 'in_progress', 'completed')),
    last_visited_at TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enrollment_id, item_id)
);

COMMENT ON TABLE course.learner_item_progress IS
    'Self-paced learner progress per enrollment item (plan 15.2).';

CREATE INDEX idx_learner_item_progress_enrollment
    ON course.learner_item_progress (enrollment_id, status);
CREATE INDEX idx_learner_item_progress_item
    ON course.learner_item_progress (item_id);

-- Platform feature flag (managed in Settings → Global platform; default off, SL tier).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_self_paced_mode BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_self_paced_mode IS
    'Enables self-paced enrollment with no instructor for self-learner courses (plan 15.2).';
