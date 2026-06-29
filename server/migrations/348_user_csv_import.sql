-- Plan 18.2: Bulk user CSV import jobs and external_id upsert key.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS bulk_csv_import_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.bulk_csv_import_enabled IS
    'Plan 18.2: Enables org-admin bulk user CSV import (/org-admin/import).';

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS external_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_org_external_id
    ON "user".users (org_id, external_id)
    WHERE external_id IS NOT NULL AND external_id <> '';

COMMENT ON COLUMN "user".users.external_id IS
    'SIS/OneRoster sourcedId for upsert matching within an org (plan 18.2).';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'import_job_status') THEN
        CREATE TYPE provisioning.import_job_status AS ENUM ('queued', 'running', 'complete', 'failed');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'import_merge_strategy') THEN
        CREATE TYPE provisioning.import_merge_strategy AS ENUM ('create_only', 'upsert', 'sync');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS provisioning.user_import_jobs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    actor_id         UUID NOT NULL REFERENCES "user".users (id),
    status           provisioning.import_job_status NOT NULL DEFAULT 'queued',
    merge_strategy   provisioning.import_merge_strategy NOT NULL,
    import_profile   TEXT NOT NULL DEFAULT 'lextures_native',
    dry_run          BOOLEAN NOT NULL DEFAULT FALSE,
    total_rows       INT,
    processed_rows   INT NOT NULL DEFAULT 0,
    error_rows       INT NOT NULL DEFAULT 0,
    created_count    INT NOT NULL DEFAULT 0,
    updated_count    INT NOT NULL DEFAULT 0,
    deactivated_count INT NOT NULL DEFAULT 0,
    skipped_count    INT NOT NULL DEFAULT 0,
    errors_jsonb     JSONB,
    cursor_row       INT NOT NULL DEFAULT 0,
    input_file_path  TEXT,
    result_file_path TEXT,
    queue_job_id     UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_import_jobs_org_status
    ON provisioning.user_import_jobs (org_id, status);

CREATE INDEX IF NOT EXISTS idx_user_import_jobs_org_created
    ON provisioning.user_import_jobs (org_id, created_at DESC);

INSERT INTO "user".provisioning_role_map (provider, external_role, app_role_id, account_type)
SELECT 'csv_import', 'teacher', r.id, 'standard' FROM "user".app_roles r WHERE r.name = 'Teacher'
UNION ALL SELECT 'csv_import', 'student', r.id, 'standard' FROM "user".app_roles r WHERE r.name = 'Student'
UNION ALL SELECT 'csv_import', 'admin', r.id, 'standard' FROM "user".app_roles r WHERE r.name = 'Teacher'
ON CONFLICT (provider, lower(external_role)) DO NOTHING;
