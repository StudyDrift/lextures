DROP TABLE IF EXISTS settings.org_email_templates;
DROP TABLE IF EXISTS settings.email_template_slots;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS email_template_editor_enabled;
