-- 227: Organization type for K-12 vs higher education feature flag
-- Defaults to 'higher-ed'; set to 'k-12' to enable grade-level features.
ALTER TABLE tenant.organizations ADD COLUMN IF NOT EXISTS org_type TEXT NOT NULL DEFAULT 'higher-ed';
ALTER TABLE tenant.organizations ADD CONSTRAINT organizations_org_type_check CHECK (org_type IN ('higher-ed', 'k-12'));
