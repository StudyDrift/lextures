-- Plan 13.7: SIS integration (PowerSchool, Infinite Campus, Skyward, Aeries).

CREATE SCHEMA IF NOT EXISTS sis;

CREATE TABLE IF NOT EXISTS sis.sis_connections (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    vendor            TEXT        NOT NULL CHECK (vendor IN ('powerschool', 'infinite_campus', 'skyward', 'aeries')),
    base_url          TEXT        NOT NULL,
    client_id_ref     TEXT        NOT NULL,
    client_secret_ref TEXT        NOT NULL,
    sync_schedule     TEXT        NOT NULL DEFAULT '0 2 * * *',
    sync_mode         TEXT        NOT NULL DEFAULT 'incremental' CHECK (sync_mode IN ('incremental', 'full')),
    active            BOOLEAN     NOT NULL DEFAULT true,
    last_sync_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sis_connections_org_idx ON sis.sis_connections (org_id);

COMMENT ON TABLE sis.sis_connections IS
    'Plan 13.7: SIS vendor connection configs per org. Credentials stored via secrets manager refs.';

CREATE TABLE IF NOT EXISTS sis.sis_sync_logs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID        NOT NULL REFERENCES sis.sis_connections (id) ON DELETE CASCADE,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at   TIMESTAMPTZ,
    status        TEXT        NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'partial', 'failed')),
    summary       JSONB,
    errors        JSONB
);

CREATE INDEX IF NOT EXISTS sis_sync_logs_connection_idx ON sis.sis_sync_logs (connection_id, started_at DESC);

COMMENT ON TABLE sis.sis_sync_logs IS
    'Plan 13.7: Audit log for each SIS sync run. FERPA-protected.';

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS external_sis_id TEXT;
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS external_sis_id TEXT;
ALTER TABLE course.course_enrollments
    ADD COLUMN IF NOT EXISTS external_sis_id TEXT;

CREATE INDEX IF NOT EXISTS users_sis_id_idx
    ON "user".users (external_sis_id) WHERE external_sis_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS courses_sis_id_idx
    ON course.courses (external_sis_id) WHERE external_sis_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS enrollments_sis_id_idx
    ON course.course_enrollments (external_sis_id) WHERE external_sis_id IS NOT NULL;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_sis_integration BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_sis_integration IS
    'Plan 13.7: Enables SIS integration endpoints and nightly sync worker.';
