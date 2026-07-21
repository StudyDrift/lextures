-- PP.1: Staff assign parent/guardian — permission + dedicated invite tokens.

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

CREATE TABLE IF NOT EXISTS "user".parent_link_invites (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    student_user_id   UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    parent_user_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    link_id           UUID NOT NULL REFERENCES "user".parent_student_links (id) ON DELETE CASCADE,
    email             TEXT NOT NULL,
    invited_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    token_hash        TEXT NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    consumed_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (link_id)
);

CREATE INDEX IF NOT EXISTS idx_parent_link_invites_token
  ON "user".parent_link_invites (token_hash)
  WHERE consumed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_parent_link_invites_email
  ON "user".parent_link_invites (org_id, email)
  WHERE consumed_at IS NULL;

COMMENT ON TABLE "user".parent_link_invites IS
  'PP.1: One-time activate tokens for pending parent/guardian link invites.';

INSERT INTO settings.email_template_slots (id, description, merge_fields, default_html, default_text)
VALUES (
    'parent_guardian_invite',
    'Invite a parent/guardian to activate their account and view a linked student',
    '{"user.first_name":"Recipient first name","user.email":"Recipient email","student.name":"Student display name","org.name":"Organization / school name","link":"Activate account link","expires_at":"Link expiration time"}'::jsonb,
    '<p>Hi {{user.first_name}},</p><p>{{org.name}} invited you to connect as a parent/guardian of <strong>{{student.name}}</strong>.</p><p><a href="{{link}}">Activate your account</a> to set a password and open the Family dashboard.</p><p>This link expires {{expires_at}}. Grades and other records are not included in this email.</p>',
    'Hi {{user.first_name}},

{{org.name}} invited you to connect as a parent/guardian of {{student.name}}.

Activate your account: {{link}}

This link expires {{expires_at}}. Grades and other records are not included in this email.'
)
ON CONFLICT (id) DO NOTHING;
