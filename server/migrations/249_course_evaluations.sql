-- Plan 14.7 — Course Evaluations (Anonymous, End-of-Term).

CREATE TABLE course.evaluation_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    questions   JSONB NOT NULL DEFAULT '[]',
    created_by  UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.evaluation_templates IS
    'Institution-defined question banks for end-of-term course evaluations (plan 14.7).';

COMMENT ON COLUMN course.evaluation_templates.questions IS
    'JSONB array of question objects: [{type, text, options, required}]. type ∈ {rating, multiple_choice, open_text}.';

CREATE INDEX idx_eval_templates_org
    ON course.evaluation_templates (org_id, created_at DESC);

CREATE TABLE course.evaluation_windows (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id       UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    template_id     UUID NOT NULL REFERENCES course.evaluation_templates (id),
    opens_at        TIMESTAMPTZ NOT NULL,
    closes_at       TIMESTAMPTZ NOT NULL,
    enrolled_count  INT NOT NULL DEFAULT 0,
    response_count  INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.evaluation_windows IS
    'Scheduled evaluation periods for a course; one active window per course at a time (plan 14.7).';

COMMENT ON COLUMN course.evaluation_windows.enrolled_count IS
    'Snapshot of enrolled student count at window creation; used for completion rate calculation.';

COMMENT ON COLUMN course.evaluation_windows.response_count IS
    'Anonymous count of submitted responses; incremented at submission time.';

CREATE INDEX idx_eval_windows_course
    ON course.evaluation_windows (course_id, opens_at DESC);

CREATE TABLE course.evaluation_responses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    window_id    UUID NOT NULL REFERENCES course.evaluation_windows (id) ON DELETE CASCADE,
    -- DELIBERATELY no user_id column — anonymity by design (plan 14.7 AC-1).
    answers      JSONB NOT NULL DEFAULT '{}',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE course.evaluation_responses IS
    'Anonymous evaluation responses. No user_id stored — responses cannot be linked to a specific student (plan 14.7).';

CREATE INDEX idx_eval_responses_window
    ON course.evaluation_responses (window_id);

-- Tracks which users have submitted for a window without linking to response rows.
-- Enables double-submission prevention and per-user status checks while preserving anonymity.
CREATE TABLE course.evaluation_submissions (
    window_id    UUID NOT NULL REFERENCES course.evaluation_windows (id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (window_id, user_id)
);

COMMENT ON TABLE course.evaluation_submissions IS
    'Records which users submitted for a window. Deliberately not joinable to evaluation_responses (plan 14.7 AC-1).';

CREATE INDEX idx_eval_submissions_window
    ON course.evaluation_submissions (window_id);

-- Feature flag for course evaluations workflow (plan 14.7).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_course_evaluations BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_course_evaluations IS
    'Enables anonymous end-of-term course evaluation system (plan 14.7). Default off; enable for HE tier.';
