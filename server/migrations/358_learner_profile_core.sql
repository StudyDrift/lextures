-- LP01 — Learner profile foundation: store, provenance model, derivation engine substrate.

CREATE SCHEMA IF NOT EXISTS learner;

CREATE TABLE learner.profiles (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL UNIQUE REFERENCES "user".users (id) ON DELETE CASCADE,
    status            TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'paused')),
    last_computed_at  TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE learner.profile_facets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id        UUID NOT NULL REFERENCES learner.profiles (id) ON DELETE CASCADE,
    facet_key         TEXT NOT NULL,
    state             TEXT NOT NULL DEFAULT 'ok'
                        CHECK (state IN ('ok', 'insufficient_data')),
    summary           JSONB NOT NULL DEFAULT '{}',
    confidence        NUMERIC(4,3) NOT NULL DEFAULT 0 CHECK (confidence BETWEEN 0 AND 1),
    computed_version  INTEGER NOT NULL DEFAULT 1,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (profile_id, facet_key)
);

CREATE TABLE learner.profile_insights (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    facet_id          UUID NOT NULL REFERENCES learner.profile_facets (id) ON DELETE CASCADE,
    insight_key       TEXT NOT NULL,
    label_i18n_key    TEXT NOT NULL,
    value             JSONB NOT NULL DEFAULT '{}',
    confidence        NUMERIC(4,3) NOT NULL DEFAULT 0 CHECK (confidence BETWEEN 0 AND 1),
    salience          INTEGER NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (facet_id, insight_key)
);

CREATE TABLE learner.profile_evidence (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    insight_id        UUID NOT NULL REFERENCES learner.profile_insights (id) ON DELETE CASCADE,
    source_kind       TEXT NOT NULL,
    source_table      TEXT NOT NULL,
    course_id         UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    observation_count INTEGER NOT NULL DEFAULT 0,
    window_start      TIMESTAMPTZ,
    window_end        TIMESTAMPTZ,
    contribution      NUMERIC(4,3),
    sample_refs       JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_lp_facets_profile ON learner.profile_facets (profile_id, facet_key);
CREATE INDEX idx_lp_insights_facet ON learner.profile_insights (facet_id, salience DESC);
CREATE INDEX idx_lp_evidence_insight ON learner.profile_evidence (insight_id);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS learner_profile_enabled BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.learner_profile_enabled IS
    'Enables the autonomous learner profile (LP epic).';