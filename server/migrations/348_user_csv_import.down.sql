DELETE FROM "user".provisioning_role_map WHERE provider = 'csv_import';

ALTER TABLE "user".provisioning_role_map
    DROP CONSTRAINT IF EXISTS provisioning_role_map_provider_check;

ALTER TABLE "user".provisioning_role_map
    ADD CONSTRAINT provisioning_role_map_provider_check
    CHECK (provider IN ('saml', 'oidc', 'scim', 'oneroster', 'clever', 'classlink'));

DROP TABLE IF EXISTS provisioning.user_import_jobs;
DROP TYPE IF EXISTS provisioning.import_merge_strategy;
DROP TYPE IF EXISTS provisioning.import_job_status;
DROP INDEX IF EXISTS "user".idx_users_org_external_id;
ALTER TABLE "user".users DROP COLUMN IF EXISTS external_id;
ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS bulk_csv_import_enabled;
