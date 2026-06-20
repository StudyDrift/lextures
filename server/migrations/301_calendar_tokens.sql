-- Plan 16.5 — Calendar feeds (iCal / CalDAV): capability tokens and rollout flag.

CREATE TABLE IF NOT EXISTS auth.calendar_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '30 days',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_calendar_tokens_user
    ON auth.calendar_tokens (user_id);

COMMENT ON TABLE auth.calendar_tokens IS
    'Short-lived capability tokens for authenticated iCal/CalDAV calendar feed URLs (plan 16.5).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_calendar_feeds BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_calendar_feeds IS
    'Enables personal and per-course iCal calendar feed subscriptions (plan 16.5).';
