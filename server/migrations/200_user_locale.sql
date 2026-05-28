-- Migration 200: User locale preference (BCP 47, plan 11.1)
ALTER TABLE "user".users
    ADD COLUMN IF NOT EXISTS locale TEXT NOT NULL DEFAULT 'en'
        CHECK (locale ~ '^[a-z]{2}(-[A-Z]{2})?$');

COMMENT ON COLUMN "user".users.locale IS 'User UI locale (BCP 47 tag). Default en.';
