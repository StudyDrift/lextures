-- Plan 3.16 — Student what-if grade projection on My Grades.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_whatif_grades BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_whatif_grades IS
    'Enables student what-if grade projection on My Grades (plan 3.16).';
