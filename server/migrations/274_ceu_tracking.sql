-- Plan 14.17 — Continuing Education (CEU) tracking with seat-time logging.

CREATE SCHEMA IF NOT EXISTS seattime;

CREATE TABLE seattime.sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    content_item_id  UUID NOT NULL REFERENCES course.course_structure_items (id) ON DELETE CASCADE,
    course_id        UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    session_token    TEXT NOT NULL,
    session_start    TIMESTAMPTZ NOT NULL,
    session_end      TIMESTAMPTZ,
    minutes_active   INT NOT NULL DEFAULT 0,
    anomaly_flag     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, content_item_id, session_token)
);

COMMENT ON TABLE seattime.sessions IS
    'Immutable seat-time audit log per content session (plan 14.17).';

CREATE INDEX idx_seattime_sessions_user_course
    ON seattime.sessions (user_id, course_id);

CREATE INDEX idx_seattime_sessions_content
    ON seattime.sessions (content_item_id);

CREATE TABLE seattime.ceu_configurations (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id            UUID NOT NULL UNIQUE REFERENCES course.courses (id) ON DELETE CASCADE,
    required_hours       NUMERIC(6, 2) NOT NULL,
    ceu_credit           NUMERIC(4, 2) NOT NULL,
    certificate_template TEXT,
    enabled              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE seattime.ceu_configurations IS
    'Per-course CEU contact-hour thresholds and certificate settings (plan 14.17).';

CREATE TABLE seattime.ceu_awards (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    course_id     UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    ceu_credit    NUMERIC(4, 2) NOT NULL,
    contact_hours NUMERIC(6, 2) NOT NULL,
    issued_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, course_id)
);

COMMENT ON TABLE seattime.ceu_awards IS
    'Issued CEU completion certificates when learners meet contact-hour thresholds (plan 14.17).';

CREATE INDEX idx_seattime_ceu_awards_user
    ON seattime.ceu_awards (user_id, issued_at DESC);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_ceu_tracking BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_ceu_tracking IS
    'Enables CEU seat-time tracking, certificates, and CE transcripts (plan 14.17).';
