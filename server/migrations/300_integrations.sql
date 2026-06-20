-- Plan 16.4 — Inbound integrations (Google Classroom, Teams Education, Canva, LTI 1.1 embeds).
-- OAuth connections, external course links, and per-provider rollout feature flags.

CREATE SCHEMA IF NOT EXISTS integrations;

-- One OAuth (2.x) connection per provider account/tenant within an org.
-- Access/refresh tokens are stored encrypted at rest (AES-256-GCM, see internal/crypto, plan 17.17).
CREATE TABLE IF NOT EXISTS integrations.oauth_connections (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    provider          TEXT NOT NULL,           -- 'google_classroom', 'microsoft_teams', 'canva'
    external_id       TEXT NOT NULL,           -- provider's account/tenant id
    access_token_enc  TEXT NOT NULL,
    refresh_token_enc TEXT NOT NULL,
    token_expires_at  TIMESTAMPTZ,
    scopes            TEXT[] NOT NULL DEFAULT '{}',
    connected_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    last_synced_at    TIMESTAMPTZ,
    last_sync_error   TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, provider, external_id),
    CHECK (provider IN ('google_classroom', 'microsoft_teams', 'canva'))
);

COMMENT ON TABLE integrations.oauth_connections IS
    'Connected third-party OAuth accounts (Google Classroom, Teams, Canva); tokens encrypted at rest (plan 16.4).';

CREATE INDEX IF NOT EXISTS idx_integrations_connections_org
    ON integrations.oauth_connections (org_id, provider);

-- Links a Lextures course to an external course/class for one-time import or recurring roster sync.
CREATE TABLE IF NOT EXISTS integrations.external_course_links (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lextures_course_id  UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    connection_id       UUID NOT NULL REFERENCES integrations.oauth_connections (id) ON DELETE CASCADE,
    external_course_id  TEXT NOT NULL,
    sync_roster         BOOLEAN NOT NULL DEFAULT true,
    sync_interval_hours SMALLINT NOT NULL DEFAULT 6,
    last_synced_at      TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (lextures_course_id, connection_id),
    CHECK (sync_interval_hours BETWEEN 1 AND 168)
);

COMMENT ON TABLE integrations.external_course_links IS
    'Maps a Lextures course to an external class for import and recurring roster sync (plan 16.4).';

CREATE INDEX IF NOT EXISTS idx_integrations_course_links_conn
    ON integrations.external_course_links (connection_id);

-- Per-provider rollout flags (separate flags, default off — see plan 16.4 §15).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_integrations_google_classroom BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_integrations_microsoft_teams BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_integrations_canva BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_integrations_google_classroom IS
    'Enables Google Classroom inbound import and roster sync (plan 16.4).';
COMMENT ON COLUMN settings.platform_app_settings.ff_integrations_microsoft_teams IS
    'Enables Microsoft Teams Education roster sync (plan 16.4).';
COMMENT ON COLUMN settings.platform_app_settings.ff_integrations_canva IS
    'Enables Canva for Education embed flow (plan 16.4).';
