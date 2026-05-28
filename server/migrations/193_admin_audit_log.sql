-- 10.11 Admin Audit Log: append-only compliance audit trail for privileged-user actions.
-- FERPA 34 CFR § 99.32 (record of disclosures), SOC 2 TSC CC7.2 (monitoring of privileged users),
-- ISO 27001:2022 Annex A.8.15 (logging), NIST SP 800-53 AU-2/AU-3/AU-9/AU-11.
-- Depends on: compliance schema (179_ferpa), "user".users (011), tenant.organizations (127),
--             settings.platform_app_settings (036).

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log (
  event_id      UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID          REFERENCES tenant.organizations(id),
  event_type    TEXT          NOT NULL,
  actor_id      UUID          NOT NULL REFERENCES "user".users(id),
  actor_ip      INET,
  user_agent    TEXT,
  target_type   TEXT,
  target_id     UUID,
  before_value  JSONB,
  after_value   JSONB,
  chain_hash    TEXT,
  "timestamp"   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
) PARTITION BY RANGE ("timestamp");

-- Monthly partitions for 2026 (automated partition management: see plan 17.4)
CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_01
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_02
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_03
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_04
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_05
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_06
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_07
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_08
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_09
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_10
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_11
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');

CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_2026_12
  PARTITION OF compliance.admin_audit_log
  FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

-- Default partition catches rows outside defined ranges (overflow; triggers automated partition job)
CREATE TABLE IF NOT EXISTS compliance.admin_audit_log_overflow
  PARTITION OF compliance.admin_audit_log DEFAULT;

-- Append-only enforcement trigger (AC-2, FR-3).
-- Fires for UPDATE/DELETE on the parent table (PostgreSQL 13+ row-level trigger inheritance).
-- Note: for DML directly on child partitions, add per-partition triggers as additional hardening.
CREATE OR REPLACE FUNCTION compliance.prevent_audit_log_modification()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'admin_audit_log is append-only; modifications are not permitted.';
END;
$$;

CREATE TRIGGER audit_log_immutable
  BEFORE UPDATE OR DELETE ON compliance.admin_audit_log
  FOR EACH ROW EXECUTE FUNCTION compliance.prevent_audit_log_modification();

-- Indexes for common query patterns (FR-8: < 2 s for 90-day date-range queries)
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_org_ts
  ON compliance.admin_audit_log(org_id, "timestamp" DESC);

CREATE INDEX IF NOT EXISTS idx_admin_audit_log_actor_ts
  ON compliance.admin_audit_log(actor_id, "timestamp" DESC);

CREATE INDEX IF NOT EXISTS idx_admin_audit_log_event_ts
  ON compliance.admin_audit_log(event_type, "timestamp" DESC);

COMMENT ON TABLE compliance.admin_audit_log IS
  'Append-only admin-action audit trail; FERPA 34 CFR § 99.32, SOC 2 CC7.2, ISO A.8.15; plan 10.11.';

-- audit:read permission — separate from admin permission (FR-6 / NFR Security)
INSERT INTO "user".permissions (permission_string, description)
VALUES ('compliance:audit:read:*', 'May query and export the admin audit log (plan 10.11).')
ON CONFLICT (permission_string) DO NOTHING;

INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM "user".app_roles r
JOIN "user".permissions p ON p.permission_string = 'compliance:audit:read:*'
WHERE r.name = 'Global Admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Feature flag: admin_audit_log_enabled defaults to TRUE in application (plan 10.11 § 15)
ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS admin_audit_log_enabled BOOLEAN;
