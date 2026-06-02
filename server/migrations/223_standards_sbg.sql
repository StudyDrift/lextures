-- Plan 13.5: Standards-based report cards (SBG). Mastery scales, domains,
-- standards taxonomy, and per-student mastery scores.

CREATE SCHEMA IF NOT EXISTS sbg;

CREATE TABLE IF NOT EXISTS sbg.standard_domains (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    code        TEXT        NOT NULL,
    name        TEXT        NOT NULL,
    grade_level TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, code)
);

CREATE INDEX IF NOT EXISTS idx_standard_domains_org
    ON sbg.standard_domains (org_id);

COMMENT ON TABLE sbg.standard_domains IS
    'Plan 13.5: Top-level grouping of standards per org (e.g., "Operations and Algebraic Thinking").';

CREATE TABLE IF NOT EXISTS sbg.standards (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id   UUID        NOT NULL REFERENCES sbg.standard_domains (id) ON DELETE CASCADE,
    code        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (domain_id, code)
);

CREATE INDEX IF NOT EXISTS idx_standards_domain
    ON sbg.standards (domain_id);

COMMENT ON TABLE sbg.standards IS
    'Plan 13.5: Individual learning standards within a domain (e.g., "CCSS.MATH.3.OA.A.1").';

CREATE TABLE IF NOT EXISTS sbg.mastery_scales (
    id      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id  UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    label   TEXT        NOT NULL,
    value   INTEGER     NOT NULL CHECK (value BETWEEN 1 AND 10),
    color   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, value)
);

CREATE INDEX IF NOT EXISTS idx_mastery_scales_org
    ON sbg.mastery_scales (org_id, value);

COMMENT ON TABLE sbg.mastery_scales IS
    'Plan 13.5: Configurable mastery level labels per org (e.g., 4=Exceeds, 3=Meets, 2=Approaching, 1=Below).';

CREATE TABLE IF NOT EXISTS sbg.mastery_scores (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id     UUID        NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    standard_id    UUID        NOT NULL REFERENCES sbg.standards (id) ON DELETE CASCADE,
    course_id      UUID        NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    grading_period TEXT        NOT NULL,
    score_value    INTEGER     NOT NULL CHECK (score_value BETWEEN 1 AND 10),
    assessed_by    UUID        REFERENCES "user".users (id),
    source         TEXT        NOT NULL CHECK (source IN ('assignment', 'quiz', 'observation')),
    source_id      UUID,
    assessed_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mastery_scores_lookup
    ON sbg.mastery_scores (student_id, standard_id, grading_period);
CREATE INDEX IF NOT EXISTS idx_mastery_scores_course_period
    ON sbg.mastery_scores (course_id, grading_period);

COMMENT ON TABLE sbg.mastery_scores IS
    'Plan 13.5: Per-student per-standard mastery evidence records. FERPA-protected education records.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_sbg_report_cards BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_sbg_report_cards IS
    'Plan 13.5: Enables standards-based grading report cards for K-12 (default false).';
