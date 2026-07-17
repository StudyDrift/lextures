-- VC.10 — Admin governance, analytics rollups, and lifecycle hooks.

CREATE TABLE IF NOT EXISTS board.org_policies (
    org_id                 UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    external_sharing       BOOLEAN NOT NULL DEFAULT FALSE,
    minor_moderation_floor BOOLEAN NOT NULL DEFAULT TRUE,
    default_attribution    TEXT NOT NULL DEFAULT 'named',
    board_cap_per_course   INTEGER,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_org_policies_attribution_check
        CHECK (default_attribution IN ('named', 'anon_to_peers', 'anonymous')),
    CONSTRAINT board_org_policies_cap_check
        CHECK (board_cap_per_course IS NULL OR board_cap_per_course >= 0)
);

COMMENT ON TABLE board.org_policies IS
    'VC.10: Org-level board policies (external sharing, minors floor, attribution, caps).';

CREATE TABLE IF NOT EXISTS board.analytics_daily (
    board_id          UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    day               DATE NOT NULL,
    card_count        INTEGER NOT NULL DEFAULT 0,
    contributor_count INTEGER NOT NULL DEFAULT 0,
    reaction_count    INTEGER NOT NULL DEFAULT 0,
    comment_count     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (board_id, day)
);

CREATE INDEX IF NOT EXISTS idx_board_analytics_daily_day
    ON board.analytics_daily (day DESC);

COMMENT ON TABLE board.analytics_daily IS
    'VC.10: Precomputed daily board engagement rollups for analytics and admin overview.';
