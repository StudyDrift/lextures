-- AC.6 down: remove serving extensions and opt-out table.

DROP TABLE IF EXISTS course.adaptive_content_optouts;

DROP INDEX IF EXISTS course.ux_ac_servings_exposure;

ALTER TABLE course.adaptation_servings
    DROP COLUMN IF EXISTS view_original_clicks,
    DROP COLUMN IF EXISTS first_viewed_at,
    DROP COLUMN IF EXISTS view_count,
    DROP COLUMN IF EXISTS content_version;
