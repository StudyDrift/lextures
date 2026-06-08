-- Plan 14.2: Course catalog & registration integration (HE).

CREATE SCHEMA IF NOT EXISTS catalog;

CREATE TABLE IF NOT EXISTS catalog.catalog_sections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    term_id         UUID NOT NULL REFERENCES tenant.terms (id) ON DELETE CASCADE,
    sis_course_id   TEXT NOT NULL,
    sis_section_id  TEXT NOT NULL,
    crn             TEXT,
    subject         TEXT NOT NULL,
    course_number   TEXT NOT NULL,
    section_number  TEXT,
    title           TEXT NOT NULL,
    credits         NUMERIC(4, 2),
    meeting_pattern JSONB,
    room            TEXT,
    department      TEXT,
    prerequisites   JSONB,
    instructor_name TEXT,
    status          TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'cancelled', 'pending')),
    lms_course_id   UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    synced_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT catalog_sections_org_sis_section_unique UNIQUE (org_id, sis_section_id)
);

CREATE INDEX IF NOT EXISTS idx_catalog_sections_org_term
    ON catalog.catalog_sections (org_id, term_id);
CREATE INDEX IF NOT EXISTS idx_catalog_sections_lms_course
    ON catalog.catalog_sections (lms_course_id) WHERE lms_course_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_catalog_sections_department
    ON catalog.catalog_sections (org_id, department) WHERE department IS NOT NULL;

COMMENT ON TABLE catalog.catalog_sections IS
    'Plan 14.2: Official SIS catalog sections synced nightly for HE institutions.';

CREATE TABLE IF NOT EXISTS catalog.student_registrations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id             UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    user_id            UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    catalog_section_id UUID NOT NULL REFERENCES catalog.catalog_sections (id) ON DELETE CASCADE,
    status             TEXT NOT NULL DEFAULT 'registered'
        CHECK (status IN ('registered', 'waitlisted', 'auditing', 'withdrawn')),
    prereq_status      JSONB,
    synced_at          TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT student_registrations_user_section_unique UNIQUE (user_id, catalog_section_id)
);

CREATE INDEX IF NOT EXISTS idx_student_registrations_user
    ON catalog.student_registrations (user_id, org_id);

COMMENT ON TABLE catalog.student_registrations IS
    'Plan 14.2: Student registration status from SIS (FERPA-protected).';

CREATE TABLE IF NOT EXISTS catalog.catalog_sync_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    connection_id   UUID REFERENCES sis.sis_connections (id) ON DELETE SET NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at     TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'success', 'partial', 'failed')),
    sections_synced INT NOT NULL DEFAULT 0,
    shells_created  INT NOT NULL DEFAULT 0,
    shells_updated  INT NOT NULL DEFAULT 0,
    errors          JSONB
);

CREATE INDEX IF NOT EXISTS idx_catalog_sync_logs_org
    ON catalog.catalog_sync_logs (org_id, started_at DESC);

COMMENT ON TABLE catalog.catalog_sync_logs IS
    'Plan 14.2: Audit log for catalog sync runs.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_catalog_integration BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_catalog_integration IS
    'Plan 14.2: Enables course catalog browse, registration status, and SIS catalog sync.';
