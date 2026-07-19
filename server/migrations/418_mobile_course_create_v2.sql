-- MOB.1: staged rollout for mobile course-create wizard parity (competency builder, Canvas entry, drafts).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_mobile_course_create_v2 BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_mobile_course_create_v2 IS
    'MOB.1: Mobile course creation wizard v2 (competency authoring, Canvas create entry, draft resume). Default OFF.';
