-- Rollback AP.2 multi-provider credential store (legacy columns retained).

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ai_tenant_allowed_providers;

ALTER TABLE settings.platform_app_settings
    DROP COLUMN IF EXISTS ai_tenant_byok_allowed;

DROP TABLE IF EXISTS settings.ai_provider_secrets;
DROP TABLE IF EXISTS settings.ai_provider_credentials;
