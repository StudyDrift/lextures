-- Plan 14.13 — Co-Curricular Transcript / Comprehensive Learner Record (CLR).

CREATE TABLE user.ccr_achievements (
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
    added_by         UUID REFERENCES "user".users (id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, achievement_type, source_id)
);

CREATE INDEX idx_ccr_achievements_user ON user.ccr_achievements (user_id, issued_at DESC);

CREATE TABLE user.ccr_documents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    generated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    consented_at  TIMESTAMPTZ,
    clr_json      JSONB NOT NULL,
    vc_proof      JSONB NOT NULL,
    pdf_key       TEXT,
    share_token   TEXT UNIQUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ccr_documents_user ON user.ccr_documents (user_id, generated_at DESC);
CREATE UNIQUE INDEX idx_ccr_documents_share_token ON user.ccr_documents (share_token) WHERE share_token IS NOT NULL;

CREATE TABLE settings.ccr_signing_config (
    id                  SERIAL PRIMARY KEY,
    issuer_did          TEXT NOT NULL,
    public_key_jwk      JSONB NOT NULL,
    private_key_cipher  BYTEA NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE user.ccr_achievements IS 'Achievements aggregated into a student CLR (plan 14.13).';
COMMENT ON TABLE user.ccr_documents IS 'Generated CLR documents with optional public share tokens (plan 14.13).';
COMMENT ON TABLE settings.ccr_signing_config IS 'Institutional Ed25519 signing key for W3C VC CLR credentials (plan 14.13).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_co_curricular_transcript BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_co_curricular_transcript IS
    'Enables co-curricular transcript / Comprehensive Learner Record generation and verification (plan 14.13).';
