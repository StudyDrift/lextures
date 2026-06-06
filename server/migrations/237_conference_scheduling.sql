-- Plan 13.12: Parent-teacher conference scheduling

CREATE SCHEMA IF NOT EXISTS conference;

CREATE TABLE IF NOT EXISTS conference.conference_availability (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    teacher_id      UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    school_id       UUID NOT NULL REFERENCES tenant.org_units (id) ON DELETE CASCADE,
    date            DATE NOT NULL,
    slot_duration   INTEGER NOT NULL CHECK (slot_duration > 0 AND slot_duration <= 60),
    gap_duration    INTEGER NOT NULL DEFAULT 0 CHECK (gap_duration >= 0 AND gap_duration <= 30),
    window_start    TIME NOT NULL,
    window_end      TIME NOT NULL,
    location        TEXT,
    video_link      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_conference_window CHECK (window_end > window_start)
);

CREATE INDEX IF NOT EXISTS conference_availability_teacher_idx
    ON conference.conference_availability (teacher_id, date);
CREATE INDEX IF NOT EXISTS conference_availability_school_idx
    ON conference.conference_availability (school_id, date);

COMMENT ON TABLE conference.conference_availability IS
    'Plan 13.12: Teacher availability windows for parent-teacher conferences.';

CREATE TABLE IF NOT EXISTS conference.conference_slots (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    availability_id     UUID NOT NULL REFERENCES conference.conference_availability (id) ON DELETE CASCADE,
    start_at            TIMESTAMPTZ NOT NULL,
    end_at              TIMESTAMPTZ NOT NULL,
    status              TEXT NOT NULL DEFAULT 'open'
                            CHECK (status IN ('open', 'booked', 'cancelled')),
    booked_by_parent    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    booked_for_child    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    booked_at           TIMESTAMPTZ,
    reminder_sent_at    TIMESTAMPTZ,
    UNIQUE (availability_id, start_at)
);

CREATE INDEX IF NOT EXISTS conference_slots_availability_idx
    ON conference.conference_slots (availability_id, start_at);
CREATE INDEX IF NOT EXISTS conference_slots_start_idx
    ON conference.conference_slots (start_at)
    WHERE status = 'open';
CREATE INDEX IF NOT EXISTS conference_slots_reminder_idx
    ON conference.conference_slots (start_at)
    WHERE status = 'booked' AND reminder_sent_at IS NULL;

COMMENT ON TABLE conference.conference_slots IS
    'Plan 13.12: Bookable parent-teacher conference slots. FERPA-protected education records.';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_conference_scheduling BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_conference_scheduling IS
    'Plan 13.12: Enables parent-teacher conference scheduling in the parent portal and teacher dashboard.';
