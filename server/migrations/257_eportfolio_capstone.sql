-- ePortfolio / Capstone artifact collection (plan 14.12).
-- Student-owned, cross-course portfolios with curated artifacts, rubric evaluation,
-- public sharing, and program outcome tagging. Artifacts are self-contained snapshots
-- so unenrolling from (or deleting) a source course never removes portfolio evidence.

CREATE SCHEMA IF NOT EXISTS portfolio;

-- A student's personal portfolio. A user may own several.
CREATE TABLE portfolio.portfolios (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    intro_text  TEXT NOT NULL DEFAULT '',
    is_public   BOOLEAN NOT NULL DEFAULT FALSE,
    -- Unguessable slug minted when the portfolio is first made public; reusable across toggles.
    public_slug TEXT UNIQUE,
    -- Ordered list of artifact ids defining display order (drag-to-reorder).
    section_order UUID[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portfolios_owner ON portfolio.portfolios (owner_id, updated_at DESC);

COMMENT ON TABLE portfolio.portfolios IS
    'Student-owned ePortfolio (plan 14.12); public_slug enables an unauthenticated read-only view.';

-- A single artifact within a portfolio. Self-contained: source references are nullable
-- and a denormalised snapshot keeps the artifact viewable after the source is gone (FR-6).
CREATE TABLE portfolio.portfolio_artifacts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id  UUID NOT NULL REFERENCES portfolio.portfolios (id) ON DELETE CASCADE,
    artifact_type TEXT NOT NULL
                    CHECK (artifact_type IN ('submission', 'upload', 'text_page', 'url')),
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    -- Source linkage (nullable; SET NULL on source removal so the snapshot survives).
    source_submission_id UUID REFERENCES course.module_assignment_submissions (id) ON DELETE SET NULL,
    source_course_id     UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    -- Denormalised file snapshot for 'submission'/'upload' artifacts.
    file_key      TEXT NOT NULL DEFAULT '',
    file_name     TEXT NOT NULL DEFAULT '',
    file_mime     TEXT NOT NULL DEFAULT '',
    -- Inline content for 'text_page' / link target for 'url'.
    text_content  TEXT NOT NULL DEFAULT '',
    external_url  TEXT NOT NULL DEFAULT '',
    -- Program / course learning outcome ids the student tagged (course.course_learning_outcomes).
    outcome_ids   UUID[] NOT NULL DEFAULT '{}',
    is_public     BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order    INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portfolio_artifacts_portfolio ON portfolio.portfolio_artifacts (portfolio_id, sort_order);
CREATE INDEX idx_portfolio_artifacts_outcomes ON portfolio.portfolio_artifacts USING GIN (outcome_ids);

COMMENT ON TABLE portfolio.portfolio_artifacts IS
    'Curated evidence within a portfolio; snapshot fields keep it viewable after the source course is gone.';

-- Rubric evaluation of an artifact by an instructor or designated reviewer.
-- One row per (artifact, reviewer); re-evaluation upserts.
CREATE TABLE portfolio.portfolio_artifact_evaluations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artifact_id   UUID NOT NULL REFERENCES portfolio.portfolio_artifacts (id) ON DELETE CASCADE,
    reviewer_id   UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    -- Rubric snapshot (criteria with levels) at evaluation time, mirroring course.module_assignments.rubric_json.
    rubric_json   JSONB,
    -- Map of criterion id -> points earned.
    scores_json   JSONB NOT NULL DEFAULT '{}',
    total_score   NUMERIC,
    feedback      TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT portfolio_artifact_evaluations_unique UNIQUE (artifact_id, reviewer_id)
);

CREATE INDEX idx_portfolio_artifact_evaluations_artifact
    ON portfolio.portfolio_artifact_evaluations (artifact_id);

COMMENT ON TABLE portfolio.portfolio_artifact_evaluations IS
    'Per-reviewer rubric evaluation of a portfolio artifact (capstone review); visible to the owner.';

-- Privacy-safe public view counter (no PII, no user id) — see plan §6 Observability.
CREATE TABLE portfolio.portfolio_views (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolio.portfolios (id) ON DELETE CASCADE,
    viewed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_portfolio_views_portfolio ON portfolio.portfolio_views (portfolio_id, viewed_at DESC);

-- Feature flag for the ePortfolio / capstone module (plan 14.12). Default off.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_eportfolio BOOLEAN;
