-- 10.14 PII redaction in operational logs: ops read permission for redaction status API.
-- GDPR Art. 5(1)(f), SOC 2 CC6.1, ISO 27001:2022 Annex A.8.15, NIST SP 800-53 AU-3/SI-12.

INSERT INTO "user".permissions (permission_string, description)
VALUES (
  'compliance:ops:redaction:read:*',
  'May read PII log redaction status and metrics (plan 10.14).'
)
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
JOIN "user".permissions p ON p.permission_string = 'compliance:ops:redaction:read:*'
WHERE r.name = 'Global Admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;
