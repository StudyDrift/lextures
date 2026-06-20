-- Plan 2.15 — Differentiated assignments: unified "assign to" targeting (everyone/section/group/student)
-- with per-target due/availability overrides and quiz extra-attempts/time-multiplier. Generalizes
-- course.section_assignment_overrides and course.enrollment_quiz_overrides into one model so there is
-- a single source of effective dates/limits (most-specific target wins: student > group > section > everyone).

CREATE TABLE course.assignment_overrides (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    structure_item_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    target_type       TEXT NOT NULL CHECK (target_type IN ('everyone', 'section', 'group', 'student')),
    -- target_id is the section id / group id / enrollment id; NULL only for target_type = 'everyone'.
    target_id         UUID,
    due_at            TIMESTAMPTZ,
    available_from    TIMESTAMPTZ,
    available_until   TIMESTAMPTZ,
    -- Quiz-only fields (plan 2.15 FR-6), layered on top of quiz.max_attempts / student_accommodations.
    extra_attempts    INTEGER CHECK (extra_attempts IS NULL OR extra_attempts >= 0),
    time_multiplier   NUMERIC(4, 2) CHECK (time_multiplier IS NULL OR time_multiplier >= 1.0),
    created_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((target_type = 'everyone') = (target_id IS NULL))
);

CREATE UNIQUE INDEX idx_assignment_overrides_unique_target
    ON course.assignment_overrides (structure_item_id, target_type, COALESCE(target_id, '00000000-0000-0000-0000-000000000000'));

CREATE INDEX idx_assignment_overrides_item ON course.assignment_overrides (structure_item_id);
CREATE INDEX idx_assignment_overrides_target ON course.assignment_overrides (target_type, target_id);

COMMENT ON TABLE course.assignment_overrides IS
    'Unified assign-to targeting (plan 2.15): one or more (everyone/section/group/student) targets per assignment/quiz item, each with optional due/availability overrides and (for quizzes) extra_attempts/time_multiplier.';

-- Backfill existing section due-date overrides (no actor recorded historically).
INSERT INTO course.assignment_overrides
    (structure_item_id, target_type, target_id, due_at, available_from, available_until, created_at)
SELECT so.structure_item_id, 'section', so.section_id, so.due_at, so.available_from, so.available_until, so.created_at
FROM course.section_assignment_overrides so;

-- Backfill existing per-enrollment quiz overrides (extra attempts / time multiplier accommodations).
INSERT INTO course.assignment_overrides
    (structure_item_id, target_type, target_id, extra_attempts, time_multiplier, created_by, created_at)
SELECT eqo.quiz_id, 'student', eqo.enrollment_id, NULLIF(eqo.extra_attempts, 0), eqo.time_multiplier, eqo.created_by, eqo.created_at
FROM course.enrollment_quiz_overrides eqo;

DROP TABLE course.section_assignment_overrides;
DROP TABLE course.enrollment_quiz_overrides;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_assign_to_overrides BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_assign_to_overrides IS
    'Enables the "Assign To" editor for section/group/student targeting with per-target due dates (plan 2.15).';
