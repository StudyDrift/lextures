-- Move previously env-only feature flags into platform settings so they are toggleable
-- in Settings → Global platform (no longer process-env driven). Idempotent ADD COLUMN.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS lrs_anonymize_actors BOOLEAN,
    ADD COLUMN IF NOT EXISTS ferpa_workflow_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS dpa_portal_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS soc2_module_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_reading_preferences BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_classroom_signals BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_library_integration BOOLEAN,
    ADD COLUMN IF NOT EXISTS diagnostic_assessments_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS srs_practice_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS irt_cat_mode_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS adaptive_learner_model_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS learner_model_ema_alpha DOUBLE PRECISION;
