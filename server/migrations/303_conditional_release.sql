-- Plan 1.11 — Conditional release & module requirements (rule-based gating).
-- Instructors define per-item completion rules, module completion modes, prerequisites,
-- and date-based unlocks; progress is tracked per enrollment.

CREATE TYPE course.module_completion_mode AS ENUM ('all_items', 'one_item', 'sequential_order');

CREATE TYPE course.item_completion_rule_type AS ENUM (
    'must_view',
    'must_mark_done',
    'must_submit',
    'must_score_at_least',
    'must_contribute'
);

CREATE TABLE course.module_requirements (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id        UUID NOT NULL UNIQUE REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    completion_mode  course.module_completion_mode NOT NULL DEFAULT 'all_items',
    unlock_at        TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.module_requirements IS
    'Per-module completion mode and optional date unlock (plan 1.11).';

CREATE INDEX idx_module_requirements_module ON course.module_requirements (module_id);

CREATE TABLE course.module_prerequisites (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id              UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    prerequisite_module_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (module_id, prerequisite_module_id),
    CHECK (module_id <> prerequisite_module_id)
);

COMMENT ON TABLE course.module_prerequisites IS
    'Module B locked until prerequisite module A is complete (plan 1.11).';

CREATE INDEX idx_module_prerequisites_module ON course.module_prerequisites (module_id);

CREATE TABLE course.item_completion_rules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id    UUID NOT NULL UNIQUE REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    rule_type  course.item_completion_rule_type NOT NULL,
    threshold  NUMERIC(5, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (rule_type = 'must_score_at_least' AND threshold IS NOT NULL AND threshold >= 0 AND threshold <= 100)
        OR (rule_type <> 'must_score_at_least' AND threshold IS NULL)
    )
);

COMMENT ON TABLE course.item_completion_rules IS
    'Per-item completion requirement for conditional release (plan 1.11).';

CREATE INDEX idx_item_completion_rules_item ON course.item_completion_rules (item_id);

CREATE TABLE course.student_item_progress (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id  UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    item_id        UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    status         TEXT NOT NULL DEFAULT 'incomplete'
        CHECK (status IN ('incomplete', 'complete')),
    met_at         TIMESTAMPTZ,
    evidence_json  JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enrollment_id, item_id)
);

COMMENT ON TABLE course.student_item_progress IS
    'Per-enrollment item requirement completion for conditional release (plan 1.11).';

CREATE INDEX idx_student_item_progress_enrollment_item
    ON course.student_item_progress (enrollment_id, item_id);

CREATE TABLE course.student_module_progress (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id  UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    module_id      UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    status         TEXT NOT NULL DEFAULT 'locked'
        CHECK (status IN ('locked', 'unlocked', 'complete')),
    unlocked_at    TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enrollment_id, module_id)
);

COMMENT ON TABLE course.student_module_progress IS
    'Per-enrollment module unlock/completion state for conditional release (plan 1.11).';

CREATE INDEX idx_student_module_progress_enrollment_module
    ON course.student_module_progress (enrollment_id, module_id);

CREATE TABLE course.module_unlock_overrides (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id  UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    module_id      UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    granted_by     UUID NOT NULL REFERENCES "user".users (id),
    granted_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enrollment_id, module_id)
);

COMMENT ON TABLE course.module_unlock_overrides IS
    'Instructor manual unlock for one student (plan 1.11); audited via admin audit log.';

CREATE INDEX idx_module_unlock_overrides_enrollment_module
    ON course.module_unlock_overrides (enrollment_id, module_id);

-- Platform feature flag (managed in Settings → Global platform; default off).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_conditional_release BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_conditional_release IS
    'Enables rule-based module requirements and conditional release (plan 1.11).';
