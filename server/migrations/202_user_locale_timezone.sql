-- Plan 11.3 / 11.4: per-user locale and IANA timezone for client-side Intl formatting.

ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS locale TEXT,
    ADD COLUMN IF NOT EXISTS timezone TEXT;

COMMENT ON COLUMN "user".users.locale IS 'BCP 47 locale tag for UI number/date formatting (e.g. en, de, fr-CA).';
COMMENT ON COLUMN "user".users.timezone IS 'IANA timezone identifier for deadline display (e.g. America/New_York, Europe/Berlin).';
