-- Companion to: 436_terms_dedupe_unique.sql
-- Data merge is not reversed; only the uniqueness guard is removed.

DROP INDEX IF EXISTS tenant.uq_terms_org_lower_name;
