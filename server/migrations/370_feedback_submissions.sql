-- Product feedback submissions (plan FB0).
-- Distinct from grading annotation feedbackmedia (migration 102).

CREATE SCHEMA IF NOT EXISTS feedback;

CREATE TABLE IF NOT EXISTS feedback.submissions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    org_id           UUID,
    message          TEXT NOT NULL,
    category         TEXT NOT NULL DEFAULT 'other',
    source           TEXT NOT NULL,
    app_version      TEXT,
    context          JSONB NOT NULL DEFAULT '{}'::jsonb,
    status           TEXT NOT NULL DEFAULT 'new',
    admin_note       TEXT,
    resolved_by      UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    resolved_at      TIMESTAMPTZ,
    idempotency_key  TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT feedback_submissions_message_len
        CHECK (char_length(message) BETWEEN 1 AND 5000),
    CONSTRAINT feedback_submissions_category_check
        CHECK (category IN ('bug', 'idea', 'question', 'praise', 'other')),
    CONSTRAINT feedback_submissions_source_check
        CHECK (source IN ('web', 'ios', 'android')),
    CONSTRAINT feedback_submissions_status_check
        CHECK (status IN ('new', 'triaged', 'in_progress', 'resolved', 'wont_fix', 'archived'))
);

COMMENT ON TABLE feedback.submissions IS
    'In-app product feedback from signed-in users (plan FB0). Not grading annotation media.';

CREATE INDEX IF NOT EXISTS idx_feedback_submissions_org_status_created
    ON feedback.submissions (org_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_feedback_submissions_created
    ON feedback.submissions (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_feedback_submissions_user
    ON feedback.submissions (user_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_feedback_submissions_user_idempotency
    ON feedback.submissions (user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Free-text admin search (plan FB0 §18).
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_feedback_submissions_message_trgm
    ON feedback.submissions USING gin (message gin_trgm_ops);

-- Platform flag (default ON handled in applyPlatformBools when NULL).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_feedback BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_feedback IS
    'Enables in-app product feedback submission (plan FB0). Default ON.';
