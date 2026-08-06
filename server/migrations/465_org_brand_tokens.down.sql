-- Companion to: 465_org_brand_tokens.sql

ALTER TABLE tenant.org_branding
  DROP CONSTRAINT IF EXISTS org_branding_brand_accent_oklch_fmt;

ALTER TABLE tenant.org_branding
  DROP COLUMN IF EXISTS brand_accent_oklch,
  DROP COLUMN IF EXISTS brand_tokens_version;
