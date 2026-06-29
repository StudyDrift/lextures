DROP TABLE IF EXISTS provisioning.user_import_jobs;
DROP TYPE IF EXISTS provisioning.import_merge_strategy;
DROP TYPE IF EXISTS provisioning.import_job_status;
DROP INDEX IF EXISTS "user".idx_users_org_external_id;
ALTER TABLE "user".users DROP COLUMN IF EXISTS external_id;
ALTER TABLE settings.platform_app_settings DROP COLUMN IF EXISTS bulk_csv_import_enabled;
