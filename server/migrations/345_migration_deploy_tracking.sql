-- Plan 17.10: correlate applied migrations with deploys for incident response.

ALTER TABLE _sqlx_migrations ADD COLUMN IF NOT EXISTS deploy_id TEXT;

COMMENT ON COLUMN _sqlx_migrations.deploy_id IS
    'Optional deploy identifier (DEPLOY_ID env) recorded when the migration was applied.';
