-- Caliper / xAPI learning event emission (plan 9.6).

CREATE SCHEMA IF NOT EXISTS analytics;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS xapi_emission_enabled BOOLEAN;

CREATE TABLE IF NOT EXISTS analytics.lrs_endpoints (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                UUID NOT NULL REFERENCES tenant.organizations(id) ON DELETE CASCADE,
    label                 TEXT NOT NULL DEFAULT '',
    endpoint_url          TEXT NOT NULL,
    auth_type             TEXT NOT NULL DEFAULT 'basic'
        CHECK (auth_type IN ('basic', 'oauth2')),
    username              TEXT,
    password_ciphertext   BYTEA,
    oauth_client_id       TEXT,
    oauth_client_secret_ciphertext BYTEA,
    oauth_token_url       TEXT,
    enabled               BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_lrs_endpoints_org ON analytics.lrs_endpoints (org_id, enabled);

CREATE TABLE IF NOT EXISTS analytics.xapi_statements (
    statement_id        UUID NOT NULL,
    actor_hash          TEXT NOT NULL,
    verb_id             TEXT NOT NULL,
    object_id           TEXT NOT NULL,
    object_type         TEXT,
    object_title        TEXT,
    result_score        REAL,
    result_success      BOOLEAN,
    context_course_id   UUID REFERENCES course.courses(id) ON DELETE SET NULL,
    stored_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    full_json           JSONB NOT NULL,
    PRIMARY KEY (statement_id, stored_at)
) PARTITION BY RANGE (stored_at);

CREATE TABLE IF NOT EXISTS analytics.xapi_statements_default
    PARTITION OF analytics.xapi_statements DEFAULT;

CREATE INDEX IF NOT EXISTS idx_xapi_statements_course_stored
    ON analytics.xapi_statements (context_course_id, stored_at DESC);

CREATE INDEX IF NOT EXISTS idx_xapi_statements_verb
    ON analytics.xapi_statements (verb_id, stored_at DESC);

CREATE TABLE IF NOT EXISTS analytics.lrs_forward_jobs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    statement_id      UUID NOT NULL,
    statement_stored_at TIMESTAMPTZ NOT NULL,
    lrs_endpoint_id   UUID NOT NULL REFERENCES analytics.lrs_endpoints(id) ON DELETE CASCADE,
    status            TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed', 'dead')),
    attempts          INTEGER NOT NULL DEFAULT 0,
    next_retry_at     TIMESTAMPTZ,
    last_http_status  INTEGER,
    last_response     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_lrs_forward_jobs_due
    ON analytics.lrs_forward_jobs (status, next_retry_at, created_at)
    WHERE status IN ('pending', 'failed');

CREATE TABLE IF NOT EXISTS analytics.lrs_dead_letter (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    statement_id      UUID NOT NULL,
    statement_stored_at TIMESTAMPTZ NOT NULL,
    lrs_endpoint_id   UUID NOT NULL REFERENCES analytics.lrs_endpoints(id) ON DELETE CASCADE,
    last_error        TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE analytics.xapi_statements IS
    'Append-only internal LRS store for xAPI 1.0.3 statements (and embedded Caliper events in full_json).';
COMMENT ON TABLE analytics.lrs_endpoints IS
    'Per-tenant external LRS forwarding targets with encrypted credentials.';
COMMENT ON TABLE analytics.lrs_forward_jobs IS
    'Async queue for POSTing statements to external LRS endpoints.';
COMMENT ON TABLE analytics.lrs_dead_letter IS
    'Statements that failed LRS forwarding after max retries.';
