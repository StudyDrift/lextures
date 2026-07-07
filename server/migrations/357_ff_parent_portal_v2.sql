-- Plan W02: Parent portal v2 sections (attendance, behavior, report cards, message teacher).

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_parent_portal_v2 BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_parent_portal_v2 IS
    'Plan W02: Enables expanded parent portal sections (attendance, behavior, report cards, message teacher). Grade title enrichment ships regardless.';
