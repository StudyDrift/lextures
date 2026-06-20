-- Plan 16.6 — Slack / Teams / Discord classroom bots.

CREATE SCHEMA IF NOT EXISTS integrations;

CREATE TABLE IF NOT EXISTS integrations.bot_connections (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id               UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    platform             TEXT NOT NULL CHECK (platform IN ('slack', 'teams', 'discord')),
    workspace_id         TEXT NOT NULL,
    workspace_name       TEXT,
    bot_token_enc        TEXT NOT NULL,
    signing_secret_enc   TEXT NOT NULL,
    webhook_subscription_id UUID REFERENCES integrations.webhook_subscriptions (id) ON DELETE SET NULL,
    settings             JSONB NOT NULL DEFAULT '{"dueSoonHours":24,"gradeChannelEnabled":false}',
    connected_by         UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, platform, workspace_id)
);

CREATE INDEX IF NOT EXISTS idx_bot_connections_org
    ON integrations.bot_connections (org_id, platform);

COMMENT ON TABLE integrations.bot_connections IS
    'Plan 16.6: Connected Slack/Teams/Discord workspaces; tokens encrypted at rest.';

CREATE TABLE IF NOT EXISTS integrations.bot_channel_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id   UUID NOT NULL REFERENCES integrations.bot_connections (id) ON DELETE CASCADE,
    course_id       UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    channel_id      TEXT NOT NULL,
    channel_name    TEXT,
    event_types     TEXT[] NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (connection_id, course_id, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_bot_channel_mappings_conn
    ON integrations.bot_channel_mappings (connection_id);

COMMENT ON TABLE integrations.bot_channel_mappings IS
    'Plan 16.6: Maps Lextures course events to platform channels.';

CREATE TABLE IF NOT EXISTS integrations.bot_user_links (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    platform         TEXT NOT NULL CHECK (platform IN ('slack', 'teams', 'discord')),
    platform_user_id TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (platform, platform_user_id),
    UNIQUE (user_id, platform)
);

CREATE INDEX IF NOT EXISTS idx_bot_user_links_user
    ON integrations.bot_user_links (user_id);

COMMENT ON TABLE integrations.bot_user_links IS
    'Plan 16.6: Links a Lextures user to their Slack/Discord/Teams identity for DMs and slash commands.';

-- Tracks assignment.due_soon notifications already sent (dedup).
CREATE TABLE IF NOT EXISTS integrations.bot_due_soon_sent (
    structure_item_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    user_id           UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    sent_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (structure_item_id, user_id)
);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_bot_slack BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_bot_teams BOOLEAN,
    ADD COLUMN IF NOT EXISTS ff_bot_discord BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_bot_slack IS
    'Plan 16.6: Enables the Lextures Slack bot (channel notifications and slash commands).';
COMMENT ON COLUMN settings.platform_app_settings.ff_bot_teams IS
    'Plan 16.6: Enables the Lextures Microsoft Teams bot.';
COMMENT ON COLUMN settings.platform_app_settings.ff_bot_discord IS
    'Plan 16.6: Enables the Lextures Discord bot.';
