-- Plan 18.5: Email template editor — slots, org versions, feature flag.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS email_template_editor_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.email_template_editor_enabled IS
    'Plan 18.5: Enables org admin email template editor APIs and UI.';

CREATE TABLE IF NOT EXISTS settings.email_template_slots (
    id           TEXT PRIMARY KEY,
    description  TEXT NOT NULL,
    merge_fields JSONB NOT NULL DEFAULT '{}'::jsonb,
    default_html TEXT NOT NULL,
    default_text TEXT NOT NULL
);

COMMENT ON TABLE settings.email_template_slots IS
    'System transactional email template slots (plan 18.5).';

CREATE TABLE IF NOT EXISTS settings.org_email_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    slot_id     TEXT NOT NULL REFERENCES settings.email_template_slots (id),
    html_body   TEXT NOT NULL,
    text_body   TEXT,
    reply_to    TEXT,
    sender_name TEXT,
    created_by  UUID REFERENCES "user".users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_active   BOOLEAN NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX IF NOT EXISTS org_email_templates_active
    ON settings.org_email_templates (org_id, slot_id)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS org_email_templates_history
    ON settings.org_email_templates (org_id, slot_id, created_at DESC);

COMMENT ON TABLE settings.org_email_templates IS
    'Per-org email template versions; one active row per org+slot (plan 18.5).';

INSERT INTO settings.email_template_slots (id, description, merge_fields, default_html, default_text)
VALUES
(
    'welcome',
    'Welcome email sent to new users',
    '{"user.first_name":"Recipient first name","user.last_name":"Recipient last name","user.email":"Recipient email","org.name":"Organization name","link":"Sign-in link"}'::jsonb,
    '<p>Welcome to <strong>{{org.name}}</strong>, {{user.first_name}}!</p><p>We are glad you joined. <a href="{{link}}">Sign in to get started</a>.</p>',
    'Welcome to {{org.name}}, {{user.first_name}}!

Sign in: {{link}}'
),
(
    'password_reset',
    'Password reset link email',
    '{"user.first_name":"Recipient first name","user.email":"Recipient email","link":"Password reset link","expires_at":"Link expiration time"}'::jsonb,
    '<p>Hi {{user.first_name}},</p><p>We received a request to reset your password. <a href="{{link}}">Reset your password</a>.</p><p>This link expires {{expires_at}}. If you did not request this, you can ignore this email.</p>',
    'Hi {{user.first_name}},

Reset your password: {{link}}

This link expires {{expires_at}}.'
),
(
    'grade_posted',
    'Grade posted notification',
    '{"user.first_name":"Recipient first name","course.title":"Course name","assignment.title":"Assignment name","link":"View grade link","unsubscribe_url":"Unsubscribe link"}'::jsonb,
    '<p>Hi {{user.first_name}},</p><p>Your grade has been posted for <strong>{{assignment.title}}</strong> in <strong>{{course.title}}</strong>.</p><p><a href="{{link}}">View your grade</a></p>',
    'Hi {{user.first_name}},

Your grade has been posted for "{{assignment.title}}" in {{course.title}}.

View your grade: {{link}}'
),
(
    'assignment_created',
    'New assignment notification',
    '{"user.first_name":"Recipient first name","course.title":"Course name","assignment.title":"Assignment name","link":"Open assignment link","unsubscribe_url":"Unsubscribe link"}'::jsonb,
    '<p>Hi {{user.first_name}},</p><p>A new assignment <strong>{{assignment.title}}</strong> was added to <strong>{{course.title}}</strong>.</p><p><a href="{{link}}">Open assignment</a></p>',
    'Hi {{user.first_name}},

A new assignment "{{assignment.title}}" was added to {{course.title}}.

Open assignment: {{link}}'
),
(
    'assignment_due_reminder',
    'Assignment due soon reminder',
    '{"user.first_name":"Recipient first name","course.title":"Course name","assignment.title":"Assignment name","assignment.due_at":"Due date/time","link":"Open assignment link","unsubscribe_url":"Unsubscribe link"}'::jsonb,
    '<p>Hi {{user.first_name}},</p><p>Your assignment <strong>{{assignment.title}}</strong> in <strong>{{course.title}}</strong> is due <strong>{{assignment.due_at}}</strong>.</p><p><a href="{{link}}">Open assignment</a></p>',
    'Hi {{user.first_name}},

Your assignment "{{assignment.title}}" in {{course.title}} is due {{assignment.due_at}}.

Open assignment: {{link}}'
),
(
    'discussion_reply',
    'Discussion reply notification',
    '{"user.first_name":"Recipient first name","course.title":"Course name","discussion.title":"Discussion topic","link":"View discussion link","unsubscribe_url":"Unsubscribe link"}'::jsonb,
    '<p>Hi {{user.first_name}},</p><p>There is a new reply in <strong>{{discussion.title}}</strong> in <strong>{{course.title}}</strong>.</p><p><a href="{{link}}">View discussion</a></p>',
    'Hi {{user.first_name}},

There is a new reply in "{{discussion.title}}" in {{course.title}}.

View discussion: {{link}}'
),
(
    'enrollment_confirmed',
    'Course enrollment confirmation',
    '{"user.first_name":"Recipient first name","course.title":"Course name","link":"Open course link","unsubscribe_url":"Unsubscribe link"}'::jsonb,
    '<p>Hi {{user.first_name}},</p><p>You are enrolled in <strong>{{course.title}}</strong>.</p><p><a href="{{link}}">Open course</a></p>',
    'Hi {{user.first_name}},

You are enrolled in {{course.title}}.

Open course: {{link}}'
)
ON CONFLICT (id) DO NOTHING;
