-- 10.9 SOC 2 Type II: access reviews, incident tracking, and vendor risk management.
-- Depends on: compliance schema (179_ferpa), "user".users (011).

-- Access review cadence (FR-2): privileged quarterly, all_production semi-annually.
CREATE TABLE IF NOT EXISTS compliance.access_reviews (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reviewer_id     UUID NOT NULL REFERENCES "user".users(id),
  review_type     TEXT NOT NULL CHECK (review_type IN ('privileged','all_production','third_party')),
  reviewed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  findings        JSONB,
  next_review_due TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_access_reviews_reviewer
  ON compliance.access_reviews(reviewer_id, reviewed_at DESC);

CREATE INDEX IF NOT EXISTS idx_access_reviews_next_due
  ON compliance.access_reviews(review_type, next_review_due);

COMMENT ON TABLE compliance.access_reviews IS
  'SOC 2 access review cadence records (TSC CC6.3); plan 10.9 FR-2.';

-- Incident response tracking (FR-4): log, contain, resolve, post-mortem.
CREATE TABLE IF NOT EXISTS compliance.incidents (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title           TEXT NOT NULL,
  severity        TEXT NOT NULL CHECK (severity IN ('P0','P1','P2','P3')),
  status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','contained','resolved','closed')),
  opened_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  resolved_at     TIMESTAMPTZ,
  post_mortem_url TEXT,
  tsc_criteria    TEXT[]
);

CREATE INDEX IF NOT EXISTS idx_incidents_status
  ON compliance.incidents(status, opened_at DESC);

CREATE INDEX IF NOT EXISTS idx_incidents_severity
  ON compliance.incidents(severity, status);

COMMENT ON TABLE compliance.incidents IS
  'SOC 2 incident response log (TSC CC7.3, CC7.4); plan 10.9 FR-4.';

-- Vendor risk register (FR-6): third-party sub-processors reviewed annually.
CREATE TABLE IF NOT EXISTS compliance.vendor_risk (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_name     TEXT NOT NULL UNIQUE,
  soc2_report_url TEXT,
  report_date     DATE,
  risk_tier       TEXT NOT NULL CHECK (risk_tier IN ('critical','high','medium','low')),
  next_review_due DATE,
  notes           TEXT
);

COMMENT ON TABLE compliance.vendor_risk IS
  'SOC 2 vendor risk register (TSC CC9.2); plan 10.9 FR-6.';

-- Permission for SOC 2 compliance admin actions (CC6.3, CC7.3).
INSERT INTO "user".permissions (permission_string, description)
VALUES ('compliance:soc2:admin:*', 'May access SOC 2 compliance admin: access reviews, incidents, vendor risk.')
ON CONFLICT (permission_string) DO NOTHING;

-- Grant the permission to Global Admin by default.
INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM "user".app_roles r
  JOIN "user".permissions p ON p.permission_string = 'compliance:soc2:admin:*'
 WHERE r.name = 'Global Admin'
ON CONFLICT DO NOTHING;

-- Feature flag: soc2_module_enabled enables the compliance admin UI and API.
ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS soc2_module_enabled BOOLEAN;
