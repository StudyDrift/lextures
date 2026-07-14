-- Pluggable transactional email providers + Amazon SES (default off).
-- SMTP remains the default backend; SES is gated by ff_email_ses.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_email_ses BOOLEAN,
    ADD COLUMN IF NOT EXISTS email_provider TEXT,
    ADD COLUMN IF NOT EXISTS ses_region TEXT,
    ADD COLUMN IF NOT EXISTS ses_from TEXT,
    ADD COLUMN IF NOT EXISTS ses_configuration_set TEXT;

COMMENT ON COLUMN settings.platform_app_settings.ff_email_ses IS
    'Enables Amazon SES as a selectable email delivery provider (default OFF). When false, email_provider=ses falls back to SMTP.';
COMMENT ON COLUMN settings.platform_app_settings.email_provider IS
    'Transactional email backend: smtp (default) or ses. Additional providers may be added later.';
COMMENT ON COLUMN settings.platform_app_settings.ses_region IS
    'AWS region for SES API calls (e.g. us-east-1). Empty falls back to env SES_REGION / AWS_REGION.';
COMMENT ON COLUMN settings.platform_app_settings.ses_from IS
    'Verified SES From address. Empty falls back to env SES_FROM or smtp_from / SMTP_FROM.';
COMMENT ON COLUMN settings.platform_app_settings.ses_configuration_set IS
    'Optional SES configuration set name for event publishing / reputation tracking.';
