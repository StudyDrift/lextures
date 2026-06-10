-- Plan 14.13 — Co-Curricular Transcript / Comprehensive Learner Record (CLR).

CREATE SCHEMA IF NOT EXISTS ccr;

CREATE TABLE ccr.achievements (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    achievement_type TEXT NOT NULL
        CHECK (achievement_type IN (
            'course_completion', 'badge', 'certificate', 'portfolio', 'extracurricular'
        )),
    source_id        UUID,
    title            TEXT NOT NULL,
    description      TEXT,
    issued_at        TIMESTAMPTZ NOT NULL,
    evidence_url     TEXT,
    outcome_tags     TEXT[] NOT NULL DEFAULT '{}',
    added_by         UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ccr.achievements IS
    'Verified achievements for CCR aggregation (plan 14.13). Manual extracurricular rows and synced badge/portfolio records.';

CREATE INDEX idx_ccr_achievements_user
    ON ccr.achievements (user_id, issued_at DESC);

CREATE TABLE ccr.documents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    clr_json     JSONB NOT NULL,
    vc_proof     JSONB NOT NULL,
    pdf_key      TEXT,
    share_token  TEXT UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE ccr.documents IS
    'Generated CLR documents signed as W3C Verifiable Credentials (plan 14.13). share_token set only when student opts into public verification.';

CREATE INDEX idx_ccr_documents_user
    ON ccr.documents (user_id, generated_at DESC);

CREATE UNIQUE INDEX idx_ccr_documents_share_token
    ON ccr.documents (share_token)
    WHERE share_token IS NOT NULL;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_co_curricular_transcript BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_co_curricular_transcript IS
    'Enables co-curricular transcript (CLR) generation, download, and public verification (plan 14.13).';
