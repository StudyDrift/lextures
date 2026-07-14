-- Companion to: 372_email_provider_ses.sql

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ff_email_ses,
    DROP COLUMN IF EXISTS email_provider,
    DROP COLUMN IF EXISTS ses_region,
    DROP COLUMN IF EXISTS ses_from,
    DROP COLUMN IF EXISTS ses_configuration_set;
