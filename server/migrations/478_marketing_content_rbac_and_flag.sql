-- MC.1 — platform gate and explicitly held marketing-content permissions.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_marketing_content BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_marketing_content IS
    'Enables the database-backed Marketing Content workspace and APIs (plan MC). Default OFF.';

INSERT INTO "user".permissions (permission_string, description) VALUES
  ('global:app:marketing-content:view', 'View the Marketing Content workspace and article list.'),
  ('global:app:marketing-content:author', 'Create and edit marketing content drafts.'),
  ('global:app:marketing-content:review', 'Approve or request changes on marketing content.'),
  ('global:app:marketing-content:publish', 'Publish, schedule, unpublish and archive marketing content.'),
  ('global:app:marketing-content:admin', 'Manage marketing content categories, authors, redirects and media.')
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
CROSS JOIN "user".permissions p
WHERE r.name = 'Global Admin'
  AND p.permission_string LIKE 'global:app:marketing-content:%'
ON CONFLICT (role_id, permission_id) DO NOTHING;

