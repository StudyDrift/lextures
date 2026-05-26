-- COPPA verifiable parental consent workflow (plan 10.2).

CREATE SCHEMA IF NOT EXISTS compliance;

ALTER TABLE "user".users
  ADD COLUMN IF NOT EXISTS coppa_minor          BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS coppa_consent_status TEXT    NOT NULL DEFAULT 'not_required'
    CHECK (coppa_consent_status IN ('not_required','pending','approved','revoked')),
  ADD COLUMN IF NOT EXISTS parent_email         TEXT,
  ADD COLUMN IF NOT EXISTS date_of_birth        DATE;

CREATE TABLE IF NOT EXISTS compliance.coppa_consents (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id              UUID        NOT NULL REFERENCES tenant.organizations (id) ON DELETE CASCADE,
  student_id          UUID        NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
  parent_email        TEXT        NOT NULL,
  consent_method      TEXT        NOT NULL
    CHECK (consent_method IN ('email_signed','school_authorization','upload','direct')),
  consent_token_hash  TEXT,
  consented_at        TIMESTAMPTZ,
  revoked_at          TIMESTAMPTZ,
  prior_record_id     UUID        REFERENCES compliance.coppa_consents (id),
  ai_features_enabled BOOLEAN     NOT NULL DEFAULT FALSE,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_coppa_consents_student
  ON compliance.coppa_consents (student_id, revoked_at NULLS FIRST);

COMMENT ON TABLE compliance.coppa_consents IS
  'Immutable consent records for COPPA 16 CFR §312.5; amendments insert a new row referencing prior_record_id (plan 10.2).';
