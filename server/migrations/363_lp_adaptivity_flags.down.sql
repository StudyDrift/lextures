ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS lp_adapt_recommendations,
    DROP COLUMN IF EXISTS lp_adapt_review,
    DROP COLUMN IF EXISTS lp_adapt_modality,
    DROP COLUMN IF EXISTS lp_adapt_tutor;