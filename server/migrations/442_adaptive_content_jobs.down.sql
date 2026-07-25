-- Down: AC.4 adaptive content jobs / budget / pause controls.

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS adaptive_content_generation_paused;

ALTER TABLE course.adaptive_content_settings
    DROP COLUMN IF EXISTS tokens_used_period,
    DROP COLUMN IF EXISTS budget_period_start,
    DROP COLUMN IF EXISTS max_prewarm_variants,
    DROP COLUMN IF EXISTS generation_paused;

DROP INDEX IF EXISTS course.idx_ac_jobs_generating_locked;
DROP INDEX IF EXISTS course.idx_ac_jobs_pickup;
DROP INDEX IF EXISTS course.ux_ac_jobs_dedupe;
DROP TABLE IF EXISTS course.adaptive_content_jobs;
