-- Plan 15.5 / 15.6 — Issued credentials (Open Badges 3.0) and LinkedIn share audit events.

CREATE SCHEMA IF NOT EXISTS credentials;

CREATE TABLE credentials.credential_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id       UUID REFERENCES course.courses (id) ON DELETE SET NULL,
    path_id         UUID REFERENCES learningpath.learning_paths (id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    description     TEXT,
    background_url  TEXT,
    logo_url        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE credentials.credential_templates IS
    'Certificate templates for course/path completion credentials (plan 15.5).';

CREATE TABLE credentials.issued_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id    UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    template_id     UUID REFERENCES credentials.credential_templates (id) ON DELETE SET NULL,
    source_type     TEXT NOT NULL CHECK (source_type IN ('course', 'path', 'ceu')),
    source_id       UUID NOT NULL,
    title           TEXT NOT NULL,
    credential_json JSONB NOT NULL,
    proof           JSONB NOT NULL,
    pdf_key         TEXT,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (recipient_id, source_type, source_id)
);

COMMENT ON TABLE credentials.issued_credentials IS
    'Issued Open Badges 3.0 / W3C VC credentials (plan 15.5).';

CREATE INDEX idx_issued_credentials_recipient
    ON credentials.issued_credentials (recipient_id, issued_at DESC);

CREATE INDEX idx_issued_credentials_source
    ON credentials.issued_credentials (source_type, source_id);

-- Platform feature flag (managed in Settings → Global platform; default off, SL tier).
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_completion_credentials BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_completion_credentials IS
    'Enables course completion certificates, Open Badges export, and LinkedIn share (plans 15.5, 15.6).';

-- 15.6: credential share analytics via user_audit.
ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_event_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_event_kind_check CHECK (
    event_kind IN (
        'course_visit',
        'content_open',
        'content_leave',
        'equation_inserted',
        'equation_editor_open',
        'credential_share_linkedin',
        'credential_share_badge_export'
    )
);

ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_structure_item_kind;
ALTER TABLE "user".user_audit DROP CONSTRAINT IF EXISTS user_audit_structure_item_kind_check;
ALTER TABLE "user".user_audit ADD CONSTRAINT user_audit_structure_item_kind_check CHECK (
    (event_kind = 'course_visit' AND structure_item_id IS NULL)
    OR (event_kind IN ('content_open', 'content_leave') AND structure_item_id IS NOT NULL)
    OR (event_kind IN ('equation_inserted', 'equation_editor_open'))
    OR (
        event_kind IN ('credential_share_linkedin', 'credential_share_badge_export')
        AND structure_item_id IS NOT NULL
    )
);