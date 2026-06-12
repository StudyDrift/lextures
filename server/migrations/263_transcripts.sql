-- Institutional transcript requests: students request official transcripts via a
-- configurable institution webhook (POST with student payload).

CREATE SCHEMA IF NOT EXISTS transcripts;

CREATE TABLE transcripts.transcript_requests (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    org_id               UUID REFERENCES tenant.organizations (id),
    status               TEXT NOT NULL DEFAULT 'queued'
                           CHECK (status IN ('queued', 'submitted', 'failed')),
    error_message        TEXT,
    webhook_response_code INT,
    requested_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transcript_requests_user
    ON transcripts.transcript_requests (user_id, requested_at DESC);

COMMENT ON TABLE transcripts.transcript_requests IS
    'Student transcript requests queued for delivery to the institution webhook.';

-- Singleton platform config for the institution webhook endpoint.
CREATE TABLE settings.transcripts_config (
    id              SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    webhook_url     TEXT,
    webhook_secret  TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO settings.transcripts_config (id) VALUES (1) ON CONFLICT DO NOTHING;

COMMENT ON TABLE settings.transcripts_config IS
    'Institution webhook URL for transcript requests (POST with student payload).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_transcripts BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_transcripts IS
    'Enables student transcript requests and institution webhook configuration. Managed in Settings → Global platform.';
