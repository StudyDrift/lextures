-- Plan 14.3 — Drop / Add / Withdrawal enrollment lifecycle (W/AU/I/NC states).

CREATE TYPE course.enrollment_state AS ENUM (
    'waitlist',
    'active',
    'dropped',
    'withdrawn',
    'audit',
    'no_credit',
    'incomplete'
);

ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS state course.enrollment_state NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS state_changed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS state_reason TEXT;

COMMENT ON COLUMN course.course_enrollments.state IS 'HE enrollment lifecycle state (plan 14.3).';
COMMENT ON COLUMN course.course_enrollments.state_changed_at IS 'Timestamp of the last state transition.';
COMMENT ON COLUMN course.course_enrollments.state_reason IS 'Optional reason for the current state.';

CREATE TABLE course.enrollment_state_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id   UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    actor_id        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    previous_state  course.enrollment_state NOT NULL,
    new_state       course.enrollment_state NOT NULL,
    reason          TEXT,
    source          TEXT NOT NULL DEFAULT 'manual'
        CHECK (source IN ('manual', 'sis_sync', 'system')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_enrollment_state_history_enrollment
    ON course.enrollment_state_history (enrollment_id, created_at DESC);

-- Term deadlines for add/drop and withdrawal enforcement (bridge until plan 14.6).
ALTER TABLE tenant.terms
    ADD COLUMN IF NOT EXISTS add_drop_deadline DATE,
    ADD COLUMN IF NOT EXISTS withdrawal_deadline DATE;

COMMENT ON COLUMN tenant.terms.add_drop_deadline IS 'Last date a student may drop with no transcript grade (plan 14.3).';
COMMENT ON COLUMN tenant.terms.withdrawal_deadline IS 'Last date a student may withdraw with a W grade (plan 14.3).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_enrollment_state_machine BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_enrollment_state_machine IS
    'Enables HE enrollment lifecycle states, gradebook former-students section, and state history (plan 14.3).';
