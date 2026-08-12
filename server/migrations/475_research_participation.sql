-- SEO.12: tenant-controlled participation in de-identified aggregate research.
CREATE TABLE IF NOT EXISTS tenant.research_participation_settings (
  org_id UUID PRIMARY KEY REFERENCES tenant.organizations(id) ON DELETE CASCADE,
  participation TEXT NOT NULL CHECK (participation IN ('opt_in', 'opt_out')),
  updated_by UUID NOT NULL REFERENCES "user".users(id),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE tenant.research_participation_settings IS
  'Explicit per-tenant aggregate research participation decision (SEO.12); absence is unresolved, never implicit consent.';

