-- AC.1 — Adaptive Content Engine foundations: per-course flag, settings, units, profiles, variants, servings, outcomes, events.

-- Per-course flag (mirrors adaptive_paths_enabled / misconception_detection_enabled).
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS adaptive_content_enabled BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN course.courses.adaptive_content_enabled IS
    'When true, the Adaptive Content Engine (ACE) may generate and serve per-learner content variants for this course.';

-- Per-course configuration (one row per course; created lazily on first PUT).
CREATE TABLE IF NOT EXISTS course.adaptive_content_settings (
    course_id UUID PRIMARY KEY REFERENCES course.courses (id) ON DELETE CASCADE,
    allowed_axes TEXT[] NOT NULL DEFAULT ARRAY['emphasis','scaffolding','reading_level','misconception'],
    default_strategy TEXT NOT NULL DEFAULT 'balanced',
    holdout_percent SMALLINT NOT NULL DEFAULT 0 CHECK (holdout_percent BETWEEN 0 AND 50),
    monthly_token_budget BIGINT NOT NULL DEFAULT 0 CHECK (monthly_token_budget >= 0),
    require_instructor_approval BOOLEAN NOT NULL DEFAULT FALSE,
    student_optout_allowed BOOLEAN NOT NULL DEFAULT TRUE,
    updated_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.adaptive_content_settings IS
    'AC.1: Per-course Adaptive Content Engine configuration. Created lazily on first settings PUT.';

-- An authorable unit: bind a target scope + its base content + pre/post assessments.
CREATE TABLE IF NOT EXISTS course.adaptive_content_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    target_kind TEXT NOT NULL CHECK (target_kind IN ('module', 'outcome')),
    target_module_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    target_outcome_id UUID REFERENCES course.course_learning_outcomes (id) ON DELETE CASCADE,
    base_content_item_id UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    pre_assessment_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE SET NULL,
    post_assessment_item_id UUID REFERENCES course.course_structure_items (id) ON DELETE SET NULL,
    allowed_axes TEXT[] NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'paused', 'archived')),
    created_by UUID NOT NULL REFERENCES "user".users (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ac_units_target_shape CHECK (
        (target_kind = 'module'  AND target_module_item_id IS NOT NULL AND target_outcome_id IS NULL) OR
        (target_kind = 'outcome' AND target_outcome_id     IS NOT NULL AND target_module_item_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_ac_units_course ON course.adaptive_content_units (course_id, status);
CREATE INDEX IF NOT EXISTS idx_ac_units_base_item ON course.adaptive_content_units (base_content_item_id);

COMMENT ON TABLE course.adaptive_content_units IS
    'AC.1: Authorable adaptive content unit binding base content + optional pre/post assessments.';

-- Per-learner adaptation decision (populated by AC.2). Kept minimal here; AC.2 adds columns.
CREATE TABLE IF NOT EXISTS course.adaptation_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    profile_signature TEXT NOT NULL,
    emphasis_mode TEXT,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_attempt_id UUID REFERENCES course.quiz_attempts (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (unit_id, enrollment_id)
);

CREATE INDEX IF NOT EXISTS idx_ac_profiles_signature ON course.adaptation_profiles (unit_id, profile_signature);

COMMENT ON TABLE course.adaptation_profiles IS
    'AC.1/AC.2: Per-learner adaptation decision for a unit. FERPA education record.';

-- Generated content variant (populated by AC.3; approvable in AC.5). Cache key = (unit, signature).
CREATE TABLE IF NOT EXISTS course.content_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    profile_signature TEXT NOT NULL,
    axes_applied TEXT[] NOT NULL DEFAULT '{}',
    variant_markdown TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    fidelity_score REAL,
    safety_flags JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft','pending_review','approved','rejected','auto_served','superseded')),
    approved_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (unit_id, profile_signature)
);

CREATE INDEX IF NOT EXISTS idx_ac_variants_unit_status ON course.content_variants (unit_id, status);

COMMENT ON TABLE course.content_variants IS
    'AC.1/AC.3: Generated content variant keyed by (unit, profile_signature).';

-- What a learner was actually served (populated by AC.6). NULL variant = base/control.
CREATE TABLE IF NOT EXISTS course.adaptation_servings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    enrollment_id UUID NOT NULL REFERENCES course.course_enrollments (id) ON DELETE CASCADE,
    profile_id UUID REFERENCES course.adaptation_profiles (id) ON DELETE SET NULL,
    variant_id UUID REFERENCES course.content_variants (id) ON DELETE SET NULL,
    was_holdout BOOLEAN NOT NULL DEFAULT FALSE,
    was_fallback BOOLEAN NOT NULL DEFAULT FALSE,
    served_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ac_servings_unit ON course.adaptation_servings (unit_id, served_at DESC);
CREATE INDEX IF NOT EXISTS idx_ac_servings_enrollment ON course.adaptation_servings (enrollment_id);

COMMENT ON TABLE course.adaptation_servings IS
    'AC.1/AC.6: Record of which variant (or base) a learner was served. FERPA education record.';

-- Loop closure: pre/post scores per serving (populated by AC.7).
CREATE TABLE IF NOT EXISTS course.adaptation_outcomes (
    serving_id UUID PRIMARY KEY REFERENCES course.adaptation_servings (id) ON DELETE CASCADE,
    pre_score_pct REAL,
    post_score_pct REAL,
    mastery_before REAL,
    mastery_after REAL,
    lift REAL,
    measured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.adaptation_outcomes IS
    'AC.1/AC.7: Pre/post scores and lift for an adaptation serving.';

-- Append-only audit for all ACE actions (settings, unit CRUD, generation, approval, serving, opt-out).
CREATE TABLE IF NOT EXISTS course.adaptive_content_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    unit_id UUID REFERENCES course.adaptive_content_units (id) ON DELETE SET NULL,
    actor_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    subject_user_id UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    detail_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ac_events_course ON course.adaptive_content_events (course_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ac_events_created_brin ON course.adaptive_content_events USING BRIN (created_at);

COMMENT ON TABLE course.adaptive_content_events IS
    'AC.1: Append-only audit log for Adaptive Content Engine actions.';
