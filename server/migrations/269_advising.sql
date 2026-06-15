-- Advising & degree-planner hooks (plan 14.14): notes, degree audit cache, config.

CREATE SCHEMA IF NOT EXISTS advising;

CREATE TABLE advising.advising_notes (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id         UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    advisor_id         UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    content            TEXT NOT NULL,
    visible_to_student BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_advising_notes_student ON advising.advising_notes (student_id, created_at DESC);

COMMENT ON TABLE advising.advising_notes IS
    'Advisor notes visible to the student and assigned advisor (plan 14.14).';

CREATE TABLE advising.degree_audit_cache (
    user_id    UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    data       JSONB NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL,
    source     TEXT NOT NULL CHECK (source IN ('degreeworks', 'stellic', 'stub'))
);

COMMENT ON TABLE advising.degree_audit_cache IS
    'Cached degree audit summary per student (4-hour TTL, plan 14.14).';

CREATE TABLE settings.advising_config (
    id                      SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    appointment_url         TEXT,
    degree_audit_provider   TEXT NOT NULL DEFAULT 'none'
        CHECK (degree_audit_provider IN ('none', 'degreeworks', 'stellic')),
    degree_audit_base_url   TEXT,
    api_credentials_ref     TEXT,
    at_risk_banner_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO settings.advising_config (id) VALUES (1) ON CONFLICT DO NOTHING;

COMMENT ON TABLE settings.advising_config IS
    'Institution advising appointment URL and degree-audit provider (plan 14.14).';

CREATE TABLE "user".advisor_student_links (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    advisor_user_id  UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    student_user_id  UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'revoked')),
    linked_by        UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    linked_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (advisor_user_id, student_user_id)
);

CREATE INDEX idx_advisor_student_links_advisor ON "user".advisor_student_links (advisor_user_id);
CREATE INDEX idx_advisor_student_links_student ON "user".advisor_student_links (student_user_id);

COMMENT ON TABLE "user".advisor_student_links IS
    'Academic advisor assignment to students (plan 14.14).';

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_advising_integration BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_advising_integration IS
    'Enables advising appointment links, degree progress widget, and advisor notes (plan 14.14).';
