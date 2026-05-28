-- 10.16 Bug bounty / responsible disclosure: vulnerability report tracking (ISO/IEC 29147, SOC 2 CC7.1).
-- Depends on: compliance schema (179_ferpa), "user".users (011), settings.platform_app_settings (036).

CREATE TABLE IF NOT EXISTS compliance.security_reports (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_handle TEXT,
  report_date     DATE NOT NULL,
  triaged_at      TIMESTAMPTZ,
  cvss_score      NUMERIC(3,1),
  severity        TEXT CHECK (severity IN ('critical','high','medium','low','informational')),
  summary         TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'triaging' CHECK (status IN ('triaging','accepted','patched','disputed','wont_fix')),
  patch_date      DATE,
  sla_met         BOOLEAN,
  bounty_paid     BOOLEAN NOT NULL DEFAULT FALSE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_security_reports_status
  ON compliance.security_reports(status, report_date DESC);

CREATE INDEX IF NOT EXISTS idx_security_reports_severity
  ON compliance.security_reports(severity, report_date DESC);

COMMENT ON TABLE compliance.security_reports IS
  'Responsible disclosure vulnerability reports for triage and patch SLA tracking; plan 10.16.';

INSERT INTO "user".permissions (permission_string, description)
VALUES ('compliance:security:admin:*', 'May manage responsible-disclosure security reports (plan 10.16).')
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM "user".app_roles r
  JOIN "user".permissions p ON p.permission_string = 'compliance:security:admin:*'
 WHERE r.name = 'Global Admin'
ON CONFLICT DO NOTHING;

ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS security_disclosure_module_enabled BOOLEAN;
