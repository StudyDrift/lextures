-- Plan 14.4 — Incomplete grade workflow with extension dates.

CREATE TABLE course.incomplete_grade_records (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id        UUID NOT NULL UNIQUE REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    granted_by           UUID NOT NULL REFERENCES "user".users (id),
    extension_deadline   DATE NOT NULL,
    outstanding_item_ids UUID[] NOT NULL DEFAULT '{}',
    notes                TEXT,
    status               TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'resolved', 'lapsed')),
    resolved_grade       TEXT,
    resolved_at          TIMESTAMPTZ,
    resolved_by          UUID REFERENCES "user".users (id),
    reminder_30d_sent_at TIMESTAMPTZ,
    reminder_7d_sent_at  TIMESTAMPTZ,
    reminder_1d_sent_at  TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incomplete_open_deadline
    ON course.incomplete_grade_records (extension_deadline)
    WHERE status = 'open';

COMMENT ON TABLE course.incomplete_grade_records IS 'HE Incomplete (I) grade records with extension deadlines (plan 14.4).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_incomplete_grade_workflow BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_incomplete_grade_workflow IS
    'Enables Incomplete grade grant/resolve workflow, reminders, and registrar report (plan 14.4).';
