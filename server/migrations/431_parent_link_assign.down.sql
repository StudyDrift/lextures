-- Rollback companion to 431_parent_link_assign.sql
-- See docs/runbooks/database-migration-rollback.md

DELETE FROM settings.email_template_slots WHERE id = 'parent_guardian_invite';

DROP TABLE IF EXISTS "user".parent_link_invites;

DELETE FROM "user".rbac_role_permissions
WHERE permission_id IN (
  SELECT id FROM "user".permissions WHERE permission_string = 'org:parent-links:assign:manage'
);

DELETE FROM "user".permissions WHERE permission_string = 'org:parent-links:assign:manage';
