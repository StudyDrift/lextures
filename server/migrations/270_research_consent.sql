-- Research / IRB consent flows (plan 14.15): consent studies and append-only consent records.

CREATE SCHEMA IF NOT EXISTS research;

CREATE TABLE research.consent_studies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    researcher_id   UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    irb_protocol    TEXT NOT NULL,
    consent_text    TEXT NOT NULL,
    data_use_desc   TEXT NOT NULL,
    target_criteria JSONB NOT NULL DEFAULT '{}'::jsonb, -- {"courseIds": [...]}
    status          TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'closed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consent_studies_org ON research.consent_studies (org_id, status);
CREATE INDEX idx_consent_studies_researcher ON research.consent_studies (researcher_id);

COMMENT ON TABLE research.consent_studies IS
    'IRB-approved research consent studies targeting student populations (plan 14.15).';

-- Append-only consent ledger. No UPDATE/DELETE; the latest row per (study, user) wins.
CREATE TABLE research.consent_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    study_id    UUID NOT NULL REFERENCES research.consent_studies (id) ON DELETE RESTRICT,
    user_id     UUID NOT NULL REFERENCES "user".users (id) ON DELETE RESTRICT,
    decision    TEXT NOT NULL CHECK (decision IN ('granted', 'declined', 'withdrawn')),
    ip_address  INET,
    user_agent  TEXT,
    hmac        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consent_records_study_user ON research.consent_records (study_id, user_id, created_at DESC);
CREATE INDEX idx_consent_records_user ON research.consent_records (user_id, created_at DESC);

COMMENT ON TABLE research.consent_records IS
    'Append-only audit ledger of consent decisions (granted/declined/withdrawn) (plan 14.15, 45 CFR 46).';

-- Enforce append-only at the database level: block UPDATE and DELETE on consent records.
CREATE OR REPLACE FUNCTION research.reject_consent_record_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'consent_records is append-only; % is not permitted', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_consent_records_append_only
    BEFORE UPDATE OR DELETE ON research.consent_records
    FOR EACH ROW EXECUTE FUNCTION research.reject_consent_record_mutation();

ALTER TABLE settings.platform_app_settings
    ADD COLUMN IF NOT EXISTS ff_research_consent BOOLEAN;

COMMENT ON COLUMN settings.platform_app_settings.ff_research_consent IS
    'Enables research / IRB consent studies, consent prompts, and gated data export (plan 14.15).';

-- Researcher capability: create and manage one's own IRB consent studies. Grant via Settings → Roles.
INSERT INTO "user".permissions (permission_string, description)
VALUES (
        'global:app:research:manage',
        'Create and manage IRB research consent studies and export consenting-participant data.'
    )
ON CONFLICT (permission_string) DO NOTHING;
