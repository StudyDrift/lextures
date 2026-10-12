-- Issue #771 — learner sign-in on a shared family device.
-- A parent marks a browser as a family device once; managed learners then pick
-- themselves from a "Who's learning?" list and enter a PIN or picture sequence.

CREATE TABLE IF NOT EXISTS "user".managed_learner_sign_ins (
    user_id          UUID PRIMARY KEY REFERENCES "user".users(id) ON DELETE CASCADE,
    method           TEXT NOT NULL CHECK (method IN ('pin', 'picture')),
    secret_hash      TEXT NOT NULL,
    failed_attempts  INT NOT NULL DEFAULT 0,
    locked_until     TIMESTAMPTZ,
    updated_by       UUID REFERENCES "user".users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE "user".managed_learner_sign_ins IS
    'Issue #771: per managed learner PIN / picture-password hash (argon2id) with failed-attempt lockout.';

CREATE TABLE IF NOT EXISTS "user".family_devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_user_id  UUID NOT NULL REFERENCES "user".users(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    label           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at    TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS family_devices_parent_active
    ON "user".family_devices (parent_user_id)
    WHERE revoked_at IS NULL;

COMMENT ON TABLE "user".family_devices IS
    'Issue #771: browsers a parent marked as a shared family device. Only a SHA-256 of the device secret is stored.';
