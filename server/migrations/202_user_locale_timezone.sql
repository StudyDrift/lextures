-- Plan 11.3 / 11.4: per-user IANA timezone for client-side Intl formatting.
-- Locale column is added in migration 200 (plan 11.1).

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS timezone TEXT;

COMMENT ON COLUMN "user".users.timezone IS 'IANA timezone identifier for deadline display (e.g. America/New_York, Europe/Berlin).';
