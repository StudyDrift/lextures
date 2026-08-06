-- UX.1 — Semantic design tokens: per-org brand accent (OKLCH) for derived ramp.

ALTER TABLE tenant.org_branding
  ADD COLUMN IF NOT EXISTS brand_accent_oklch text,
  ADD COLUMN IF NOT EXISTS brand_tokens_version int NOT NULL DEFAULT 1;

ALTER TABLE tenant.org_branding
  DROP CONSTRAINT IF EXISTS org_branding_brand_accent_oklch_fmt;

ALTER TABLE tenant.org_branding
  ADD CONSTRAINT org_branding_brand_accent_oklch_fmt
  CHECK (brand_accent_oklch IS NULL OR brand_accent_oklch ~ '^oklch\(');

COMMENT ON COLUMN tenant.org_branding.brand_accent_oklch IS
  'UX.1 org brand accent as OKLCH string (e.g. oklch(0.55 0.18 264)); NULL = product accent.';
COMMENT ON COLUMN tenant.org_branding.brand_tokens_version IS
  'Bumped when brand accent changes so clients can invalidate cached CSS vars.';
