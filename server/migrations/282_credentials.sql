-- Plan 15.5 — Certificates of completion (Open Badges 3.0 / W3C VC).

CREATE SCHEMA IF NOT EXISTS credentials;

CREATE TABLE credentials.credential_templates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id      UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    path_id        UUID REFERENCES learningpath.learning_paths (id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    description    TEXT,
    background_url TEXT,
    logo_url       TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE credentials.credential_templates IS
    'Optional per-course or per-path certificate branding (plan 15.5). NULL course_id and path_id rows are platform defaults.';

CREATE UNIQUE INDEX idx_credential_templates_course
    ON credentials.credential_templates (course_id)
    WHERE course_id IS NOT NULL;

CREATE UNIQUE INDEX idx_credential_templates_path
    ON credentials.credential_templates (path_id)
    WHERE path_id IS NOT NULL;

CREATE TABLE credentials.issued_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    template_id     UUID REFERENCES credentials.credential_templates (id) ON DELETE SET NULL,
    source_type     TEXT NOT NULL CHECK (source_type IN ('course', 'path', 'ceu')),
    source_id       UUID NOT NULL,
    credential_json JSONB NOT NULL,
    proof           JSONB NOT NULL,
    pdf_key         TEXT,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (recipient_id, source_type, source_id)
);

COMMENT ON TABLE credentials.issued_credentials IS
    'Signed Open Badges 3.0 / W3C Verifiable Credentials issued on course or path completion (plan 15.5).';

CREATE INDEX idx_issued_credentials_recipient
    ON credentials.issued_credentials (recipient_id, issued_at DESC);

CREATE INDEX idx_issued_credentials_source
    ON credentials.issued_credentials (source_type, source_id);

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_completion_credentials BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_completion_credentials IS
    'Enables Open Badges 3.0 completion certificates for self-paced courses and learning paths (plan 15.5).';