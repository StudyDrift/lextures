-- Plan PS.2: per-user pinned settings for the assignment and quiz editor panels.

CREATE TABLE IF NOT EXISTS settings.user_pinned_settings (
    user_id      UUID   NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    surface      TEXT   NOT NULL,
    setting_keys TEXT[] NOT NULL DEFAULT '{}',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, surface),
    CONSTRAINT ups_surface_check  CHECK (surface IN ('assignment', 'quiz')),
    -- Key shape/length is enforced in application code (ValidateKeys); Postgres CHECK
    -- cannot use unnest()/subqueries (SQLSTATE 0A000).
    CONSTRAINT ups_max_pins_check CHECK (cardinality(setting_keys) <= 12)
);

COMMENT ON TABLE settings.user_pinned_settings IS
    'Plan PS.2: ordered per-user pinned setting keys for the assignment/quiz editor settings panels.';
COMMENT ON COLUMN settings.user_pinned_settings.setting_keys IS
    'Ordered array; index 0 renders first. Keys come from the web settings registry (PS.1) and are shape-validated, not membership-validated, server-side.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_pinned_settings BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.platform_app_settings.ff_pinned_settings IS
    'Plan PS.2 (ff_pinned_settings): enables per-user pinned settings in the assignment/quiz editors. Default false; flip true after QA sign-off.';
