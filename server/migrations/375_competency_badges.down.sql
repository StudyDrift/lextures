-- Rollback plan B1 competency micro-badges.

ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_event_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_event_kind_check CHECK (
    event_kind IN (
        'course_visit',
        'content_open',
        'content_leave',
        'equation_inserted',
        'equation_editor_open',
        'credential_share_linkedin',
        'credential_share_badge_export'
    )
);

ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_structure_item_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_structure_item_kind_check CHECK (
    (event_kind = 'course_visit' AND structure_item_id IS NULL)
    OR (event_kind IN ('content_open', 'content_leave') AND structure_item_id IS NOT NULL)
    OR (event_kind IN ('equation_inserted', 'equation_editor_open'))
    OR (
        event_kind IN ('credential_share_linkedin', 'credential_share_badge_export')
        AND structure_item_id IS NOT NULL
    )
);

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS badges_default_public;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_competency_badges;

DROP TABLE IF EXISTS "user".user_badge_handle_history;
DROP TABLE IF EXISTS "user".user_badge_profiles;
DROP TABLE IF EXISTS badges.badge_page_views;
DROP TABLE IF EXISTS badges.awarded_badges;
DROP TABLE IF EXISTS badges.badge_definitions;
DROP TABLE IF EXISTS badges.reserved_handles;
DROP SCHEMA IF EXISTS badges;
