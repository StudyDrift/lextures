-- 10.5 SDPC / National-DPA Template: DPA versions, acceptance records, and data inventory.
-- Depends on: compliance schema (179_ferpa), "user".users (011), tenant.organizations (127).

CREATE TABLE IF NOT EXISTS compliance.dpa_versions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    version_str  TEXT        NOT NULL UNIQUE,
    template_url TEXT        NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Acceptance records are append-only; no UPDATE or DELETE permitted.
CREATE TABLE IF NOT EXISTS compliance.dpa_acceptances (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID        NOT NULL REFERENCES tenant.organizations(id),
    dpa_version_id UUID        NOT NULL REFERENCES compliance.dpa_versions(id),
    accepted_by    UUID        NOT NULL REFERENCES "user".users(id),
    accepted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address     INET,
    UNIQUE (org_id, dpa_version_id)
);

-- Deny UPDATE and DELETE to keep acceptance records append-only.
CREATE OR REPLACE RULE dpa_acceptances_no_update AS
    ON UPDATE TO compliance.dpa_acceptances DO INSTEAD NOTHING;
CREATE OR REPLACE RULE dpa_acceptances_no_delete AS
    ON DELETE TO compliance.dpa_acceptances DO INSTEAD NOTHING;

CREATE INDEX IF NOT EXISTS idx_dpa_acceptances_org
    ON compliance.dpa_acceptances(org_id, dpa_version_id);

CREATE TABLE IF NOT EXISTS compliance.data_inventory (
    id                         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    element_name               TEXT    NOT NULL,
    category                   TEXT    NOT NULL,
    purpose                    TEXT    NOT NULL,
    legal_basis                TEXT    NOT NULL,
    retention_days             INTEGER,
    shared_with_sub_processors BOOLEAN NOT NULL DEFAULT FALSE,
    sub_processor_names        TEXT[]  NOT NULL DEFAULT '{}',
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed the initial SDPC NDPA version.
INSERT INTO compliance.dpa_versions (version_str, template_url, effective_at, notes)
VALUES ('2026-01-01', 'https://static.lextures.com/legal/ndpa-2026-01-01.pdf', '2026-01-01T00:00:00Z',
        'Initial SDPC National Data Privacy Agreement baseline (NDPA v3 compatible).')
ON CONFLICT (version_str) DO NOTHING;

-- Seed the Lextures data inventory per SDPC NDPA exhibit requirements.
INSERT INTO compliance.data_inventory
    (element_name, category, purpose, legal_basis, retention_days, shared_with_sub_processors, sub_processor_names)
VALUES
    ('Email address',          'identity',    'Account authentication and communication',              'contract',           365*7, FALSE, '{}'),
    ('Display name',           'identity',    'Personalization and course roster display',             'contract',           365*7, FALSE, '{}'),
    ('First name',             'identity',    'Personalization and course roster display',             'contract',           365*7, FALSE, '{}'),
    ('Last name',              'identity',    'Personalization and course roster display',             'contract',           365*7, FALSE, '{}'),
    ('Date of birth',          'identity',    'Age verification for COPPA compliance',                'legal_obligation',   365*7, FALSE, '{}'),
    ('Assignment text',        'academic',    'AI quiz generation and rubric feedback',               'contract',           365*2, TRUE,  ARRAY['OpenRouter']),
    ('Student response',       'academic',    'Automated grading and AI tutoring feedback',           'contract',           365*2, TRUE,  ARRAY['OpenRouter']),
    ('Quiz attempt scores',    'academic',    'Adaptive learning path and progress reporting',        'contract',           365*7, FALSE, '{}'),
    ('Course enrollment',      'academic',    'Course access and grade reporting',                    'contract',           365*7, FALSE, '{}'),
    ('Learning event log',     'behavioral',  'Adaptive path recommendations and analytics',          'legitimate_interests',365*2,FALSE, '{}'),
    ('IP address (hashed)',    'technical',   'Audit logging and fraud prevention',                   'legal_obligation',   365,   FALSE, '{}'),
    ('Session token',          'technical',   'Authenticated session management',                     'contract',           1,     FALSE, '{}')
ON CONFLICT DO NOTHING;

COMMENT ON TABLE compliance.dpa_versions IS
    'Versioned DPA/NDPA templates with S3/R2 PDF path; plan 10.5.';
COMMENT ON TABLE compliance.dpa_acceptances IS
    'Append-only DPA acceptance records per org per version; plan 10.5.';
COMMENT ON TABLE compliance.data_inventory IS
    'SDPC-compatible data inventory listing student data elements collected; plan 10.5.';
