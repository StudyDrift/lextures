-- 10.1 FERPA Workflow: directory opt-out, record-access requests, consent records, disclosure log.
-- Depends on: "user".users (011), org.organizations (127).

CREATE SCHEMA IF NOT EXISTS compliance;

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS ferpa_directory_opt_out BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS ferpa_eligible_student   BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS compliance.ferpa_record_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES org.organizations(id),
    student_id      UUID NOT NULL REFERENCES "user".users(id),
    requester_id    UUID NOT NULL REFERENCES "user".users(id),
    request_type    TEXT NOT NULL CHECK (request_type IN ('inspect','amend','hearing')),
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','denied','completed')),
    amendment_field TEXT,
    amendment_value TEXT,
    notes           TEXT,
    archive_path    TEXT,
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at          TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ferpa_record_requests_student
    ON compliance.ferpa_record_requests(student_id, status);

CREATE INDEX IF NOT EXISTS idx_ferpa_record_requests_org_status
    ON compliance.ferpa_record_requests(org_id, status, requested_at DESC);

CREATE TABLE IF NOT EXISTS compliance.ferpa_consent_records (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES org.organizations(id),
    student_id   UUID NOT NULL REFERENCES "user".users(id),
    granted_by   UUID NOT NULL REFERENCES "user".users(id),
    recipient    TEXT NOT NULL,
    purpose      TEXT NOT NULL,
    data_fields  TEXT[] NOT NULL DEFAULT '{}',
    consented_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ferpa_consent_student
    ON compliance.ferpa_consent_records(student_id, revoked_at);

CREATE TABLE IF NOT EXISTS compliance.ferpa_disclosure_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES org.organizations(id),
    accessor_id     UUID NOT NULL REFERENCES "user".users(id),
    student_id      UUID NOT NULL REFERENCES "user".users(id),
    data_type       TEXT NOT NULL,
    authority_claim TEXT NOT NULL,
    recipient       TEXT,
    logged_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ferpa_disclosure_log_student
    ON compliance.ferpa_disclosure_log(student_id, logged_at DESC);

CREATE INDEX IF NOT EXISTS idx_ferpa_disclosure_log_org
    ON compliance.ferpa_disclosure_log(org_id, logged_at DESC);

-- Feature flag.
ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ferpa_workflow_enabled BOOLEAN;
