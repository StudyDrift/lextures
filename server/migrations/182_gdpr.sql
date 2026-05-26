-- 10.3 GDPR / UK GDPR: DSAR workflow, consent management, RoPA, and erasure.
-- Depends on: compliance schema (179_ferpa), "user".users (011), tenant.organizations (127).

CREATE SCHEMA IF NOT EXISTS compliance;

-- GDPR consent records (Article 6 & 7): one row per user per purpose per consent event.
CREATE TABLE IF NOT EXISTS compliance.gdpr_consents (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    purpose         TEXT        NOT NULL,
    lawful_basis    TEXT        NOT NULL CHECK (lawful_basis IN (
                                    'consent','contract','legal_obligation',
                                    'vital_interests','legitimate_interests')),
    consent_version TEXT        NOT NULL,
    granted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    withdrawn_at    TIMESTAMPTZ,
    ip_hash         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gdpr_consents_user
    ON compliance.gdpr_consents(user_id, purpose, withdrawn_at NULLS FIRST);

-- Data Subject Access Requests (Articles 15, 17, 20).
-- due_at defaults to 30 days after creation per GDPR Art. 12(3) statutory deadline.
CREATE TABLE IF NOT EXISTS compliance.dsar_requests (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID        REFERENCES tenant.organizations(id),
    user_id          UUID        NOT NULL REFERENCES "user".users(id),
    request_type     TEXT        NOT NULL CHECK (request_type IN (
                                     'access','erasure','portability',
                                     'rectification','restriction','objection')),
    status           TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN (
                                     'pending','in_progress','completed','rejected')),
    archive_url      TEXT,
    archive_expires_at TIMESTAMPTZ,
    rejection_reason TEXT,
    requested_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at           TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    completed_at     TIMESTAMPTZ,
    actioned_by      UUID        REFERENCES "user".users(id)
);

CREATE INDEX IF NOT EXISTS idx_dsar_requests_due
    ON compliance.dsar_requests(due_at) WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_dsar_requests_user
    ON compliance.dsar_requests(user_id, status, requested_at DESC);

-- Records of Processing Activities (Article 30).
CREATE TABLE IF NOT EXISTS compliance.ropa_entries (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID        NOT NULL REFERENCES tenant.organizations(id),
    activity_name    TEXT        NOT NULL,
    purpose          TEXT        NOT NULL,
    lawful_basis     TEXT        NOT NULL,
    data_categories  TEXT[]      NOT NULL DEFAULT '{}',
    data_subjects    TEXT[]      NOT NULL DEFAULT '{}',
    retention_period TEXT        NOT NULL,
    sub_processors   TEXT[]      NOT NULL DEFAULT '{}',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ropa_entries_org
    ON compliance.ropa_entries(org_id, updated_at DESC);

COMMENT ON TABLE compliance.gdpr_consents IS
    'Granular GDPR consent records per user per processing purpose (Art. 6 & 7); plan 10.3.';
COMMENT ON TABLE compliance.dsar_requests IS
    'Data Subject Access, Erasure, and Portability requests (Art. 15, 17, 20); plan 10.3.';
COMMENT ON TABLE compliance.ropa_entries IS
    'Records of Processing Activities per tenant (Art. 30); plan 10.3.';
