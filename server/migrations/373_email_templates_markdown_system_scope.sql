-- ET-1: Markdown source column, system-scope templates, and system-email slots.
-- Extends plan 18.5 (email template editor).

-- Markdown source alongside compiled HTML (ET-1).
ALTER TABLE settings.email_template_slots
    ADD COLUMN IF NOT EXISTS default_markdown TEXT NOT NULL DEFAULT '';

ALTER TABLE settings.org_email_templates
    ADD COLUMN IF NOT EXISTS source_markdown TEXT;

COMMENT ON COLUMN settings.email_template_slots.default_markdown IS
    'ET-1: Canonical Markdown default for the slot (compiled to default_html by ET-2).';
COMMENT ON COLUMN settings.org_email_templates.source_markdown IS
    'ET-1: Per-org Markdown source; html_body is the compiled artifact (ET-2).';

-- System (platform) scope: org-less overrides, one active row per slot.
CREATE TABLE IF NOT EXISTS settings.system_email_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id         TEXT NOT NULL REFERENCES settings.email_template_slots (id),
    source_markdown TEXT NOT NULL,
    html_body       TEXT NOT NULL,
    text_body       TEXT,
    reply_to        TEXT,
    sender_name     TEXT,
    created_by      UUID REFERENCES "user".users (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_active       BOOLEAN NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX IF NOT EXISTS system_email_templates_active
    ON settings.system_email_templates (slot_id)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS system_email_templates_history
    ON settings.system_email_templates (slot_id, created_at DESC);

COMMENT ON TABLE settings.system_email_templates IS
    'ET-1: Platform-wide (org-less) email template versions; one active row per slot.';

-- New system-email slots (magic link + COPPA). ON CONFLICT keeps existing rows.
INSERT INTO settings.email_template_slots
    (id, description, merge_fields, default_html, default_text, default_markdown)
VALUES
(
    'magic_link',
    'Passwordless sign-in link',
    '{"user.first_name":"Recipient first name","link":"One-time sign-in link","expires_at":"Link expiration"}'::jsonb,
    '<p>Sign in to your account without a password.</p><p><a href="{{link}}">Sign in now</a></p><p>This link works once and expires {{expires_at}}. If you did not request it, ignore this email.</p>',
    'Sign in to your account without a password.

Sign in: {{link}}

This link works once and expires {{expires_at}}. If you did not request it, ignore this email.',
    'Sign in to your account without a password.

[Sign in now]({{link}})

This link works once and expires {{expires_at}}. If you did not request it, ignore this email.'
),
(
    'coppa_consent',
    'COPPA parent consent notice (16 CFR §312.4(c))',
    '{"student.name":"Student first name","org.name":"Organization name","link":"Consent review link","expires_at":"Consent link expiration"}'::jsonb,
    '<h2>Parent permission required</h2><p>A school account has been created for <strong>{{student.name}}</strong> on {{org.name}}.</p><p>Under the Children''s Online Privacy Protection Act (COPPA), we need your permission before activating the account.</p><ul><li><strong>What we collect:</strong> first name, school ID, course progress, quiz responses</li><li><strong>How we use it:</strong> deliver coursework and track learning progress</li><li><strong>Third-party sharing:</strong> none without your consent</li></ul><p><a href="{{link}}">Review &amp; give permission</a></p><p>This link expires {{expires_at}}. If unexpected, ignore this — no account activates without approval.</p>',
    'Parent permission required

A school account has been created for {{student.name}} on {{org.name}}.

Under the Children''s Online Privacy Protection Act (COPPA), we need your permission before activating the account.

What we collect: first name, school ID, course progress, quiz responses
How we use it: deliver coursework and track learning progress
Third-party sharing: none without your consent

Review & give permission: {{link}}

This link expires {{expires_at}}. If unexpected, ignore this — no account activates without approval.',
    '## Parent permission required

A school account has been created for **{{student.name}}** on {{org.name}}.

Under COPPA we need your permission before activating the account.

- **What we collect:** first name, school ID, course progress, quiz responses
- **How we use it:** deliver coursework and track learning progress
- **Third-party sharing:** none without your consent

[Review & give permission]({{link}})

This link expires {{expires_at}}. If unexpected, ignore this — no account activates without approval.'
),
(
    'coppa_consent_confirmation',
    'COPPA consent confirmed',
    '{"student.name":"Student first name","org.name":"Organization name"}'::jsonb,
    '<p>You have given permission for <strong>{{student.name}}</strong> to use {{org.name}}.</p><p>Their account is now active. You can manage privacy settings or revoke permission any time by contacting your school.</p>',
    'You have given permission for {{student.name}} to use {{org.name}}.

Their account is now active. You can manage privacy settings or revoke permission any time by contacting your school.',
    'You have given permission for **{{student.name}}** to use {{org.name}}.

Their account is now active. You can manage privacy settings or revoke permission any time by contacting your school.'
)
ON CONFLICT (id) DO NOTHING;

-- Backfill default_markdown for the seven existing 18.5 slots (only when still empty).
UPDATE settings.email_template_slots SET default_markdown =
  'Welcome to **{{org.name}}**, {{user.first_name}}!

We are glad you joined. [Sign in to get started]({{link}}).'
  WHERE id = 'welcome' AND default_markdown = '';

UPDATE settings.email_template_slots SET default_markdown =
  'Hi {{user.first_name}},

We received a request to reset your password. [Reset your password]({{link}}).

This link expires {{expires_at}}. If you did not request this, ignore this email.'
  WHERE id = 'password_reset' AND default_markdown = '';

UPDATE settings.email_template_slots SET default_markdown =
  'Hi {{user.first_name}},

Your grade has been posted for **{{assignment.title}}** in **{{course.title}}**.

[View your grade]({{link}})'
  WHERE id = 'grade_posted' AND default_markdown = '';

UPDATE settings.email_template_slots SET default_markdown =
  'Hi {{user.first_name}},

A new assignment **{{assignment.title}}** was added to **{{course.title}}**.

[Open assignment]({{link}})'
  WHERE id = 'assignment_created' AND default_markdown = '';

UPDATE settings.email_template_slots SET default_markdown =
  'Hi {{user.first_name}},

Your assignment **{{assignment.title}}** in **{{course.title}}** is due **{{assignment.due_at}}**.

[Open assignment]({{link}})'
  WHERE id = 'assignment_due_reminder' AND default_markdown = '';

UPDATE settings.email_template_slots SET default_markdown =
  'Hi {{user.first_name}},

There is a new reply in **{{discussion.title}}** in **{{course.title}}**.

[View discussion]({{link}})'
  WHERE id = 'discussion_reply' AND default_markdown = '';

UPDATE settings.email_template_slots SET default_markdown =
  'Hi {{user.first_name}},

You are enrolled in **{{course.title}}**.

[Open course]({{link}})'
  WHERE id = 'enrollment_confirmed' AND default_markdown = '';

-- Plan 18.8 seat alert (seeded in 354) — ensure non-empty default_markdown (AC-1).
UPDATE settings.email_template_slots SET default_markdown =
  'Your organization **{{orgName}}** has reached {{thresholdPct}}% of its licensed seats ({{usedSeats}} / {{maxSeats}}).

Contact your Lextures representative to purchase additional seats before new users are blocked.'
  WHERE id = 'seat_utilization_alert' AND default_markdown = '';
