-- Plan 16.10: Zapier / Make.com connector — webhook settings and feature flag.

ALTER TABLE integrations.webhook_subscriptions
    ADD COLUMN IF NOT EXISTS settings JSONB NOT NULL DEFAULT '{}';

COMMENT ON COLUMN integrations.webhook_subscriptions.settings IS
    'Plan 16.10: Connector metadata (source=zapier|make, includePII, etc.).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_zapier_connector BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_zapier_connector IS
    'Plan 16.10: Enables Zapier/Make REST-hook webhook subscriptions via automation connectors.';
