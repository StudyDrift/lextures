-- PP.1: Fix permission to four-segment form required by ValidatePermissionString.
-- Safe if 431 already seeded the incorrect three-segment string.

INSERT INTO "user".permissions (permission_string, description)
VALUES (
  'org:parent-links:assign:manage',
  'Search students and assign parent/guardian links (invite if account missing)'
)
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
CROSS JOIN "user".permissions p
WHERE r.name = 'Global Admin'
  AND p.permission_string = 'org:parent-links:assign:manage'
ON CONFLICT (role_id, permission_id) DO NOTHING;

DELETE FROM "user".rbac_role_permissions
WHERE permission_id IN (
  SELECT id FROM "user".permissions WHERE permission_string = 'org:parent-links:manage'
);

DELETE FROM "user".permissions WHERE permission_string = 'org:parent-links:manage';
