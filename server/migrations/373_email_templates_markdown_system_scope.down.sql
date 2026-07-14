-- ET-1 down: remove system scope, markdown columns, and the three new slot rows only.
-- The seven original 18.5 slots remain intact.

DROP TABLE IF EXISTS settings.system_email_templates;

DELETE FROM settings.email_template_slots
WHERE id IN ('magic_link', 'coppa_consent', 'coppa_consent_confirmation');

ALTER TABLE settings.org_email_templates
    DROP COLUMN IF EXISTS source_markdown;

ALTER TABLE settings.email_template_slots
    DROP COLUMN IF EXISTS default_markdown;
