-- AC.8 down: remove governance tables/columns.

ALTER TABLE course.courses
    DROP COLUMN IF EXISTS adaptive_content_quarantined_reason,
    DROP COLUMN IF EXISTS adaptive_content_quarantined;

ALTER TABLE course.adaptive_content_units
    DROP COLUMN IF EXISTS quarantined_by,
    DROP COLUMN IF EXISTS quarantined_at,
    DROP COLUMN IF EXISTS quarantined_reason,
    DROP COLUMN IF EXISTS quarantined;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS adaptive_content_kill_switch,
    DROP COLUMN IF EXISTS adaptive_content_org_enabled;

DROP TABLE IF EXISTS analytics.adaptive_content_fairness;
DROP TABLE IF EXISTS course.adaptive_content_contests;
