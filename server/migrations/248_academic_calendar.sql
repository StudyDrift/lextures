-- Plan 14.6 — Academic Calendar Awareness (Drop Dates, Finals, No-Class Days).

CREATE TYPE tenant.calendar_event_type AS ENUM (
    'term_start', 'term_end', 'add_drop_deadline', 'withdrawal_deadline',
    'finals_start', 'finals_end', 'no_class_day', 'holiday', 'custom'
);

CREATE TABLE tenant.academic_calendar_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    term_id      UUID REFERENCES tenant.terms (id) ON DELETE SET NULL,
    event_type   tenant.calendar_event_type NOT NULL,
    event_name   TEXT NOT NULL,
    start_date   DATE NOT NULL,
    end_date     DATE,
    all_day      BOOLEAN NOT NULL DEFAULT TRUE,
    notes        TEXT,
    sis_id       TEXT,
    created_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT calendar_event_end_after_start CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_calendar_events_org_term
    ON tenant.academic_calendar_events (org_id, term_id, start_date);

CREATE INDEX idx_calendar_events_org_date
    ON tenant.academic_calendar_events (org_id, start_date);

COMMENT ON TABLE tenant.academic_calendar_events IS
    'Institutional academic calendar events per organization and term (plan 14.6).';

COMMENT ON COLUMN tenant.academic_calendar_events.event_type IS
    'Categorises the event for deadline enforcement and display (term_start, add_drop_deadline, etc.).';

COMMENT ON COLUMN tenant.academic_calendar_events.sis_id IS
    'Opaque SIS identifier used to deduplicate imports from Banner/Workday.';

-- Feature flag for academic calendar (plan 14.6).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_academic_calendar BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_academic_calendar IS
    'Enables academic calendar events, dashboard upcoming-dates panel, and iCal feed (plan 14.6).';
