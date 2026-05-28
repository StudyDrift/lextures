-- 10.10 ISO/IEC 27001:2022 ISMS and ISO/IEC 27701:2019 PIMS program tracking.
-- Depends on: compliance schema (179_ferpa), "user".users (011).

CREATE TABLE IF NOT EXISTS compliance.iso_audit_findings (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_cycle       TEXT        NOT NULL,
    finding_type      TEXT        NOT NULL CHECK (finding_type IN ('nonconformity', 'observation', 'opportunity')),
    iso_clause        TEXT        NOT NULL,
    description       TEXT        NOT NULL,
    status            TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'closed')),
    corrective_action TEXT,
    due_date          DATE,
    closed_at         TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_iso_audit_findings_status
    ON compliance.iso_audit_findings(status, due_date NULLS LAST);

CREATE INDEX IF NOT EXISTS idx_iso_audit_findings_cycle
    ON compliance.iso_audit_findings(audit_cycle, created_at DESC);

CREATE TABLE IF NOT EXISTS compliance.risk_register (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    risk_title     TEXT        NOT NULL,
    likelihood     INTEGER     NOT NULL CHECK (likelihood BETWEEN 1 AND 5),
    impact         INTEGER     NOT NULL CHECK (impact BETWEEN 1 AND 5),
    treatment      TEXT        NOT NULL CHECK (treatment IN ('mitigate', 'accept', 'transfer', 'avoid')),
    residual_score INTEGER     GENERATED ALWAYS AS (likelihood * impact) STORED,
    owner_id       UUID        REFERENCES "user".users(id) ON DELETE SET NULL,
    review_date    DATE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_register_score
    ON compliance.risk_register(residual_score DESC, created_at DESC);

CREATE TABLE IF NOT EXISTS compliance.security_training_completions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    training_year INTEGER    NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, training_year)
);

CREATE INDEX IF NOT EXISTS idx_security_training_year
    ON compliance.security_training_completions(training_year, completed_at DESC);

CREATE TABLE IF NOT EXISTS compliance.supplier_reviews (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_name      TEXT        NOT NULL UNIQUE,
    review_status    TEXT        NOT NULL DEFAULT 'pending'
                         CHECK (review_status IN ('pending', 'approved', 'rejected')),
    certificate_type TEXT        CHECK (certificate_type IN ('iso27001', 'soc2', 'questionnaire', 'other')),
    certificate_url  TEXT,
    reviewed_at      TIMESTAMPTZ,
    next_review_due  DATE,
    notes            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_reviews_due
    ON compliance.supplier_reviews(next_review_due) WHERE review_status = 'approved';

CREATE TABLE IF NOT EXISTS compliance.iso_soa_controls (
    control_id              TEXT        PRIMARY KEY,
    theme                   TEXT        NOT NULL CHECK (theme IN ('organizational', 'people', 'physical', 'technological')),
    title                   TEXT        NOT NULL,
    status                  TEXT        NOT NULL DEFAULT 'planned'
                                CHECK (status IN ('implemented', 'planned', 'excluded')),
    exclusion_justification TEXT,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance.isms_program_status (
    id                    SMALLINT    PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    scope_statement       TEXT        NOT NULL DEFAULT 'Lextures LMS SaaS: web application, Go API, PostgreSQL data store, object storage, and supporting AWS infrastructure in iac/production/.',
    iso27001_status       TEXT        NOT NULL DEFAULT 'in_progress'
                              CHECK (iso27001_status IN ('not_started', 'in_progress', 'stage1', 'certified')),
    iso27001_cert_url     TEXT,
    iso27001_last_audit   DATE,
    iso27701_status       TEXT        NOT NULL DEFAULT 'in_progress'
                              CHECK (iso27701_status IN ('not_started', 'in_progress', 'certified')),
    soa_last_review       DATE,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO compliance.isms_program_status (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS iso_isms_enabled BOOLEAN;

COMMENT ON TABLE compliance.iso_audit_findings IS
    'ISO 27001 internal and external audit findings with corrective actions; plan 10.10.';
COMMENT ON TABLE compliance.risk_register IS
    'ISMS information-security risk register; plan 10.10.';
COMMENT ON TABLE compliance.security_training_completions IS
    'Annual ISO 27001-aligned security awareness training completions; plan 10.10 FR-6.';
COMMENT ON TABLE compliance.supplier_reviews IS
    'Sub-processor security review records (ISO 27001 A.5.22); plan 10.10 FR-7.';
COMMENT ON TABLE compliance.iso_soa_controls IS
    'ISO/IEC 27001:2022 Annex A Statement of Applicability control status; plan 10.10.';
COMMENT ON TABLE compliance.isms_program_status IS
    'Singleton ISMS/PIMS program metadata for trust center and certification tracking; plan 10.10.';

INSERT INTO "user".permissions (permission_string, description)
VALUES ('compliance:iso:admin:*', 'ISO 27001/27701 ISMS program administration (plan 10.10).')
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
JOIN "user".permissions p ON p.permission_string = 'compliance:iso:admin:*'
WHERE r.name = 'Global Admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;
