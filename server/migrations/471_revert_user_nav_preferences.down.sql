-- Re-apply UX.7 schema if 471 is rolled back.

CREATE TABLE IF NOT EXISTS settings.user_nav_preferences (
    user_id    UUID        NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    scope      TEXT        NOT NULL,
    pinned     JSONB       NOT NULL DEFAULT '[]'::jsonb,
    hidden     JSONB       NOT NULL DEFAULT '[]'::jsonb,
    collapsed  JSONB       NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, scope),
    CONSTRAINT user_nav_preferences_scope_check CHECK (
        scope = 'global'
        OR scope = 'settings'
        OR scope = 'admin'
        OR scope ~ '^course:[A-Za-z0-9._~-]+$'
        OR scope ~ '^course-settings:[A-Za-z0-9._~-]+$'
    )
);

CREATE INDEX IF NOT EXISTS user_nav_preferences_user_idx
    ON settings.user_nav_preferences (user_id);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_navigation_v2 BOOLEAN NOT NULL DEFAULT FALSE;
