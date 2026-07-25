-- AC.4 — Adaptive content generation pipeline: async jobs, budget cache, pause controls.

CREATE TABLE IF NOT EXISTS course.adaptive_content_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    profile_signature TEXT NOT NULL,
    content_version INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','generating','done','failed','dead_letter','canceled')),
    attempts SMALLINT NOT NULL DEFAULT 0,
    priority SMALLINT NOT NULL DEFAULT 0,
    run_after TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_by TEXT,
    locked_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One in-flight/complete job per (unit, signature, content_version) → dedupe.
CREATE UNIQUE INDEX IF NOT EXISTS ux_ac_jobs_dedupe
    ON course.adaptive_content_jobs (unit_id, profile_signature, content_version)
    WHERE status IN ('pending','generating','done');

CREATE INDEX IF NOT EXISTS idx_ac_jobs_pickup
    ON course.adaptive_content_jobs (priority DESC, run_after ASC)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_ac_jobs_generating_locked
    ON course.adaptive_content_jobs (locked_at)
    WHERE status = 'generating';

COMMENT ON TABLE course.adaptive_content_jobs IS
    'AC.4: Async generation jobs for adaptive content variants. Claimed via SELECT … FOR UPDATE SKIP LOCKED.';

-- Per-course generation controls + period accounting cache.
ALTER TABLE course.adaptive_content_settings
    ADD COLUMN IF NOT EXISTS generation_paused BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS max_prewarm_variants SMALLINT NOT NULL DEFAULT 12
        CHECK (max_prewarm_variants >= 0 AND max_prewarm_variants <= 100),
    ADD COLUMN IF NOT EXISTS budget_period_start DATE,
    ADD COLUMN IF NOT EXISTS tokens_used_period BIGINT NOT NULL DEFAULT 0
        CHECK (tokens_used_period >= 0);

COMMENT ON COLUMN course.adaptive_content_settings.generation_paused IS
    'AC.4: When true, the pipeline does not start new generations for this course.';
COMMENT ON COLUMN course.adaptive_content_settings.max_prewarm_variants IS
    'AC.4: Cap on pre-warm jobs per unit (default 12 ≈ AC.2 signature cap).';
COMMENT ON COLUMN course.adaptive_content_settings.budget_period_start IS
    'AC.4: Start of the current token accounting period (month); cache key for tokens_used_period.';
COMMENT ON COLUMN course.adaptive_content_settings.tokens_used_period IS
    'AC.4: Cached sum of adaptive_content tokens this period; reconciled from analytics.ai_usage_log.';

-- Platform-wide pipeline switch (distinct from AC.1 kill-switch which also blocks serving).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS adaptive_content_generation_paused BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.platform_app_settings.adaptive_content_generation_paused IS
    'AC.4: Ops/admin pause for adaptive content *generation* only (serving existing cache still allowed).';
