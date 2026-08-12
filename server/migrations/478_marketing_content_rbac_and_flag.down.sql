DELETE FROM "user".rbac_role_permissions
WHERE permission_id IN (
    SELECT id FROM "user".permissions WHERE permission_string LIKE 'global:app:marketing-content:%'
);

DELETE FROM "user".permissions WHERE permission_string LIKE 'global:app:marketing-content:%';

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_marketing_content;
