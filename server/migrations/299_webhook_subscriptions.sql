-- Plan 16.3: Outbound webhooks — subscriptions and delivery log.

CREATE SCHEMA IF NOT EXISTS integrations;

CREATE TABLE IF NOT EXISTS integrations.webhook_subscriptions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    label            TEXT NOT NULL,
    endpoint_url     TEXT NOT NULL,
    signing_key_enc  TEXT NOT NULL,
    event_types      TEXT[] NOT NULL,
    active           BOOLEAN NOT NULL DEFAULT true,
    paused_at        TIMESTAMPTZ,
    tls_skip_verify  BOOLEAN NOT NULL DEFAULT false,
    created_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS webhook_subscriptions_org_idx
    ON integrations.webhook_subscriptions (org_id, created_at DESC);

COMMENT ON TABLE integrations.webhook_subscriptions IS
    'Plan 16.3: Org-scoped outbound webhook endpoint subscriptions.';

CREATE TABLE IF NOT EXISTS integrations.webhook_deliveries (
    id               BIGSERIAL PRIMARY KEY,
    subscription_id  UUID NOT NULL REFERENCES integrations.webhook_subscriptions (id) ON DELETE CASCADE,
    event_type       TEXT NOT NULL,
    event_id         UUID NOT NULL,
    payload_hash     TEXT NOT NULL,
    attempt_count    SMALLINT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'delivered', 'failed', 'dead_lettered')),
    last_http_status SMALLINT,
    last_response    TEXT,
    latency_ms       INTEGER,
    next_retry_at    TIMESTAMPTZ,
    delivered_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS webhook_deliveries_subscription_idx
    ON integrations.webhook_deliveries (subscription_id, created_at DESC);

CREATE INDEX IF NOT EXISTS webhook_deliveries_retry_idx
    ON integrations.webhook_deliveries (status, next_retry_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX IF NOT EXISTS webhook_deliveries_event_idx
    ON integrations.webhook_deliveries (event_id);

COMMENT ON TABLE integrations.webhook_deliveries IS
    'Plan 16.3: Delivery attempts for outbound webhooks (90-day retention policy).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_webhooks BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_webhooks IS
    'Plan 16.3: Enables outbound webhook subscriptions, delivery, and admin UI.';
