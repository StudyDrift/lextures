-- Plan 13.2: Daily attendance (per-period, daily elementary, state-reportable).

CREATE TABLE IF NOT EXISTS course.attendance_codes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    code       TEXT NOT NULL,
    label      TEXT NOT NULL,
    state_code TEXT,
    category   TEXT NOT NULL DEFAULT 'present'
                   CHECK (category IN ('present', 'absent', 'tardy', 'other')),
    UNIQUE (org_id, code)
);

CREATE INDEX IF NOT EXISTS idx_attendance_codes_org
    ON course.attendance_codes (org_id);

COMMENT ON TABLE course.attendance_codes IS
    'Plan 13.2: Configurable per-org attendance codes (CALPADS-compatible, pluggable for any US state).';

CREATE TABLE IF NOT EXISTS course.attendance_records (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID        NOT NULL REFERENCES "user".users (id)           ON DELETE CASCADE,
    section_id  UUID        NOT NULL REFERENCES course.course_sections (id) ON DELETE CASCADE,
    school_id   UUID        REFERENCES tenant.org_units (id)                ON DELETE SET NULL,
    date        DATE        NOT NULL,
    period      TEXT,
    code_id     UUID        NOT NULL REFERENCES course.attendance_codes (id),
    note        TEXT,
    recorded_by UUID        REFERENCES "user".users (id),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS attendance_unique
    ON course.attendance_records (student_id, section_id, date, COALESCE(period, ''));
CREATE INDEX IF NOT EXISTS attendance_section_date
    ON course.attendance_records (section_id, date);
CREATE INDEX IF NOT EXISTS attendance_student_date
    ON course.attendance_records (student_id, date);

COMMENT ON TABLE course.attendance_records IS
    'Plan 13.2: Per-student per-section attendance records (daily or per-period mode). FERPA-protected education record.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_attendance BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_attendance IS
    'Plan 13.2: Enables daily/per-period attendance recording for K-12 schools.';
