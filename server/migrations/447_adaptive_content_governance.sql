-- AC.8: Governance, safety, fairness, privacy & compliance for Adaptive Content Engine.
-- Contests, fairness audit aggregates, org toggle, unit quarantine, durable kill-switch.

CREATE TABLE IF NOT EXISTS course.adaptive_content_contests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    unit_id UUID NOT NULL REFERENCES course.adaptive_content_units (id) ON DELETE CASCADE,
    serving_id UUID REFERENCES course.adaptation_servings (id) ON DELETE SET NULL,
    student_user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    reason TEXT,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'reviewed', 'resolved', 'dismissed')),
    resolved_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ac_contests_unit
    ON course.adaptive_content_contests (unit_id, status);

CREATE INDEX IF NOT EXISTS idx_ac_contests_course_status
    ON course.adaptive_content_contests (course_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ac_contests_student
    ON course.adaptive_content_contests (student_user_id);

COMMENT ON TABLE course.adaptive_content_contests IS
    'AC.8: Student/guardian contest that an adaptation seems wrong; routes to instructor review.';

-- Fairness audit results (aggregate, small-cell suppressed; per course × dimension × group).
CREATE TABLE IF NOT EXISTS analytics.adaptive_content_fairness (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    dimension TEXT NOT NULL,
    group_label TEXT NOT NULL,
    n INTEGER NOT NULL,
    mean_fidelity REAL,
    coverage_pct REAL,
    mean_lift REAL,
    disparity_flag BOOLEAN NOT NULL DEFAULT FALSE,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, dimension, group_label)
);

CREATE INDEX IF NOT EXISTS idx_ac_fairness_course
    ON analytics.adaptive_content_fairness (course_id, dimension);

CREATE INDEX IF NOT EXISTS idx_ac_fairness_disparity
    ON analytics.adaptive_content_fairness (course_id)
    WHERE disparity_flag = TRUE;

COMMENT ON TABLE analytics.adaptive_content_fairness IS
    'AC.8: Aggregate fairness audit cells (language/grade_band/section/accommodation) with suppression.';

-- Org-level ACE governance toggle (admin visibility/disable; NULL = no opinion).
-- Durable kill-switch OR'd with env ADAPTIVE_CONTENT_KILL_SWITCH.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS adaptive_content_org_enabled BOOLEAN,
    ADD COLUMN IF NOT EXISTS adaptive_content_kill_switch BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN settings.platform_app_settings.adaptive_content_org_enabled IS
    'AC.8: NULL = no org opinion; false = affirmative org-wide ACE disable; true = org allows ACE.';
COMMENT ON COLUMN settings.platform_app_settings.adaptive_content_kill_switch IS
    'AC.8: Durable admin kill-switch; OR''d with ADAPTIVE_CONTENT_KILL_SWITCH env.';

-- Incident quarantine on units (serving stops instantly → base only).
ALTER TABLE course.adaptive_content_units
    ADD COLUMN IF NOT EXISTS quarantined BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS quarantined_reason TEXT,
    ADD COLUMN IF NOT EXISTS quarantined_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS quarantined_by UUID REFERENCES "user".users (id) ON DELETE SET NULL;

COMMENT ON COLUMN course.adaptive_content_units.quarantined IS
    'AC.8: When true, ResolveServing always returns base for this unit.';

-- Course-level quarantine flag for incident cascade.
ALTER TABLE course.courses
    ADD COLUMN IF NOT EXISTS adaptive_content_quarantined BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS adaptive_content_quarantined_reason TEXT;

COMMENT ON COLUMN course.courses.adaptive_content_quarantined IS
    'AC.8: Course-wide ACE quarantine; serving falls back to base for all units.';
