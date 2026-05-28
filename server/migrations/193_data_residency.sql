-- 10.12 Data Residency: per-tenant region pinning, immutability trigger, access log, and feature flag.
-- Depends on: tenant.organizations (127), compliance schema (179_ferpa), "user".users (011).

-- Strengthen the data_region column with a valid-region CHECK constraint.
-- The column exists since migration 127 with default 'us-east-1'; v1 ships us-east and eu-west.
ALTER TABLE tenant.organizations
    DROP CONSTRAINT IF EXISTS organizations_data_region_check;

ALTER TABLE tenant.organizations
    ADD CONSTRAINT organizations_data_region_check
        CHECK (data_region IN ('us-east','eu-west','au-east','ca-central','us-east-1'));

-- Backfill legacy 'us-east-1' values to canonical 'us-east'.
UPDATE tenant.organizations
SET data_region = 'us-east'
WHERE data_region = 'us-east-1';

-- Update default to canonical value.
ALTER TABLE tenant.organizations
    ALTER COLUMN data_region SET DEFAULT 'us-east';

-- Now drop the legacy value from the constraint.
ALTER TABLE tenant.organizations
    DROP CONSTRAINT organizations_data_region_check;

ALTER TABLE tenant.organizations
    ADD CONSTRAINT organizations_data_region_check
        CHECK (data_region IN ('us-east','eu-west','au-east','ca-central'));

-- Immutability trigger: once data_region is set it cannot be changed without a migration workflow.
CREATE OR REPLACE FUNCTION tenant.prevent_data_region_change()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF OLD.data_region IS DISTINCT FROM NEW.data_region THEN
    RAISE EXCEPTION 'data_region cannot be changed after provisioning without a migration workflow'
      USING ERRCODE = 'P0002';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER organizations_data_region_immutable
  BEFORE UPDATE ON tenant.organizations
  FOR EACH ROW EXECUTE FUNCTION tenant.prevent_data_region_change();

COMMENT ON TRIGGER organizations_data_region_immutable ON tenant.organizations IS
  'FR-4 (plan 10.12): data_region is immutable after provisioning.';

-- Cross-region access attempt log (FR-5, AC-5).
CREATE TABLE IF NOT EXISTS compliance.data_residency_access_log (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id         UUID        NOT NULL REFERENCES tenant.organizations(id) ON DELETE CASCADE,
  org_region     TEXT        NOT NULL,
  requested_from TEXT        NOT NULL,
  event_type     TEXT        NOT NULL DEFAULT 'cross_region_access_blocked',
  request_path   TEXT,
  actor_id       UUID        REFERENCES "user".users(id) ON DELETE SET NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_data_residency_log_org
  ON compliance.data_residency_access_log(org_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_data_residency_log_event
  ON compliance.data_residency_access_log(event_type, created_at DESC);

COMMENT ON TABLE compliance.data_residency_access_log IS
  'Cross-region data-access attempt log; plan 10.12 FR-5, AC-5.';

-- Permission for data residency admin actions.
INSERT INTO "user".permissions (permission_string, description)
VALUES ('compliance:data-residency:admin:*', 'May access data residency compliance admin: region info and access log (plan 10.12).')
ON CONFLICT (permission_string) DO NOTHING;

-- Grant permission to Global Admin by default.
INSERT INTO "user".rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM "user".app_roles r
  JOIN "user".permissions p ON p.permission_string = 'compliance:data-residency:admin:*'
 WHERE r.name = 'Global Admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Feature flag: data_residency_enabled enables the per-tenant region enforcement UI and API.
ALTER TABLE settings.platform_app_settings
  ADD COLUMN IF NOT EXISTS data_residency_enabled BOOLEAN;
