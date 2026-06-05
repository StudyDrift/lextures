-- Plan 13.10: District-wide and emergency broadcast messages.

CREATE SCHEMA IF NOT EXISTS broadcast;

CREATE TABLE IF NOT EXISTS broadcast.broadcasts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    school_id     UUID,
    sender_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    type          TEXT NOT NULL DEFAULT 'announcement'
        CHECK (type IN ('announcement', 'emergency')),
    audience      JSONB NOT NULL DEFAULT '{}'::jsonb,
    subject       TEXT NOT NULL,
    body          TEXT NOT NULL,
    scheduled_at  TIMESTAMPTZ,
    sent_at       TIMESTAMPTZ,
    status        TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'queued', 'sent')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS broadcasts_org_idx    ON broadcast.broadcasts (org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS broadcasts_sender_idx ON broadcast.broadcasts (sender_id);
CREATE INDEX IF NOT EXISTS broadcasts_type_idx   ON broadcast.broadcasts (org_id, type);

COMMENT ON TABLE broadcast.broadcasts IS
    'Plan 13.10: District/school broadcast messages. type=emergency requires acknowledgement.';

CREATE TABLE IF NOT EXISTS broadcast.broadcast_receipts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    broadcast_id    UUID NOT NULL REFERENCES broadcast.broadcasts (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    channel         TEXT NOT NULL
        CHECK (channel IN ('in_app', 'email', 'push')),
    delivered_at    TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    UNIQUE (broadcast_id, user_id, channel)
);

CREATE INDEX IF NOT EXISTS broadcast_receipts_user_idx
    ON broadcast.broadcast_receipts (user_id, channel)
    WHERE acknowledged_at IS NULL;

COMMENT ON TABLE broadcast.broadcast_receipts IS
    'Plan 13.10: Per-user delivery + acknowledgement record for each broadcast/channel.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_broadcasts BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_broadcasts IS
    'Plan 13.10: Enables district/school broadcast messages and emergency acknowledgement.';
