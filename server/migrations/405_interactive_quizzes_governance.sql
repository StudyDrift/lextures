-- IQ.11 — Admin governance, quotas, analytics rollups, review queue, and lifecycle.

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS iq_max_concurrent_games INTEGER,
    ADD COLUMN IF NOT EXISTS iq_max_players_per_game INTEGER NOT NULL DEFAULT 300,
    ADD COLUMN IF NOT EXISTS iq_max_kits_per_course INTEGER,
    ADD COLUMN IF NOT EXISTS iq_retention_days INTEGER NOT NULL DEFAULT 365,
    ADD COLUMN IF NOT EXISTS iq_guest_join_policy TEXT NOT NULL DEFAULT 'disabled',
    ADD COLUMN IF NOT EXISTS iq_default_mode TEXT NOT NULL DEFAULT 'live_classic',
    ADD COLUMN IF NOT EXISTS iq_default_leaderboard_privacy TEXT NOT NULL DEFAULT 'names',
    ADD COLUMN IF NOT EXISTS iq_ai_generation_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS iq_ai_generations_per_day INTEGER;

-- Conservative concurrent-games default until horizontal fan-out lands.
UPDATE settings.platform_app_settings
SET iq_max_concurrent_games = 50
WHERE iq_max_concurrent_games IS NULL;

DO $$ BEGIN
    ALTER TABLE settings.platform_app_settings
        ADD CONSTRAINT platform_iq_guest_join_policy_chk
        CHECK (iq_guest_join_policy IN ('disabled', 'teacher_mediated', 'open'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE settings.platform_app_settings
        ADD CONSTRAINT platform_iq_default_mode_chk
        CHECK (iq_default_mode IN ('live_classic', 'team', 'student_paced', 'homework'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE settings.platform_app_settings
        ADD CONSTRAINT platform_iq_leaderboard_privacy_chk
        CHECK (iq_default_leaderboard_privacy IN ('names', 'anon_to_peers', 'anonymous'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE settings.platform_app_settings
        ADD CONSTRAINT platform_iq_max_players_chk
        CHECK (iq_max_players_per_game > 0);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE settings.platform_app_settings
        ADD CONSTRAINT platform_iq_retention_days_chk
        CHECK (iq_retention_days > 0);
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON COLUMN settings.platform_app_settings.iq_max_concurrent_games IS
    'IQ.11: Max concurrent live games (lobby/running/paused) tenant-wide. NULL = unlimited.';
COMMENT ON COLUMN settings.platform_app_settings.iq_max_players_per_game IS
    'IQ.11: Max active players per live game. Default 300.';
COMMENT ON COLUMN settings.platform_app_settings.iq_max_kits_per_course IS
    'IQ.11: Max non-archived kits per course. NULL = unlimited.';
COMMENT ON COLUMN settings.platform_app_settings.iq_retention_days IS
    'IQ.11: Days to retain ended-session responses before anonymisation/deletion. Default 365.';
COMMENT ON COLUMN settings.platform_app_settings.iq_guest_join_policy IS
    'IQ.11: disabled | teacher_mediated | open — platform default guest-join policy.';
COMMENT ON COLUMN settings.platform_app_settings.iq_default_mode IS
    'IQ.11: Default game mode for new hosts.';
COMMENT ON COLUMN settings.platform_app_settings.iq_default_leaderboard_privacy IS
    'IQ.11: Default leaderboard privacy for new games.';
COMMENT ON COLUMN settings.platform_app_settings.iq_ai_generation_enabled IS
    'IQ.11: Section default for AI generation enablement (also gated by ff_iq_ai_generation).';
COMMENT ON COLUMN settings.platform_app_settings.iq_ai_generations_per_day IS
    'IQ.11: Max AI generation jobs per org per day. NULL = unlimited (AI gateway budgets still apply).';

-- Org-unit / org overrides (bounded by platform). Uses tenant.organizations (platform org model).
CREATE TABLE IF NOT EXISTS quizgame.org_settings (
    org_id      UUID PRIMARY KEY REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    overrides   JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE quizgame.org_settings IS
    'IQ.11: Org-scoped Live Quiz overrides (quotas/defaults), bounded by platform settings.';

CREATE TABLE IF NOT EXISTS quizgame.review_queue (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind         TEXT NOT NULL,
    kit_id       UUID REFERENCES quizgame.kits (id) ON DELETE CASCADE,
    session_id   UUID REFERENCES quizgame.sessions (id) ON DELETE CASCADE,
    detail       JSONB NOT NULL DEFAULT '{}'::jsonb,
    status       TEXT NOT NULL DEFAULT 'pending',
    reviewer_id  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    reason       TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at  TIMESTAMPTZ,
    CONSTRAINT quizgame_review_queue_kind_chk
        CHECK (kind IN ('catalog_submission', 'reported_content')),
    CONSTRAINT quizgame_review_queue_status_chk
        CHECK (status IN ('pending', 'approved', 'rejected', 'actioned')),
    CONSTRAINT quizgame_review_queue_target_chk
        CHECK (kit_id IS NOT NULL OR session_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_review_pending
    ON quizgame.review_queue (status, created_at ASC)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_quizgame_review_kit
    ON quizgame.review_queue (kit_id)
    WHERE kit_id IS NOT NULL;

COMMENT ON TABLE quizgame.review_queue IS
    'IQ.11: Moderation/catalog review queue for public-catalog submissions and reported content.';

CREATE TABLE IF NOT EXISTS quizgame.usage_daily (
    day            DATE NOT NULL,
    org_id         UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    course_id      UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    games          INTEGER NOT NULL DEFAULT 0,
    players        INTEGER NOT NULL DEFAULT 0,
    answers        INTEGER NOT NULL DEFAULT 0,
    guest_players  INTEGER NOT NULL DEFAULT 0,
    enrolled_players INTEGER NOT NULL DEFAULT 0,
    ai_cost_cents  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (day, org_id, course_id)
);

CREATE INDEX IF NOT EXISTS idx_quizgame_usage_daily_org_day
    ON quizgame.usage_daily (org_id, day DESC);

COMMENT ON TABLE quizgame.usage_daily IS
    'IQ.11: Precomputed daily Live Quiz usage rollups for admin analytics.';
