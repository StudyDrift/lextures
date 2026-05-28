-- 10.13 Encryption at Rest: application-level encrypted columns and key registry.

CREATE SCHEMA IF NOT EXISTS secrets;

CREATE TABLE IF NOT EXISTS secrets.encryption_keys (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key_id      TEXT NOT NULL UNIQUE,
  key_version INTEGER NOT NULL DEFAULT 1,
  algorithm   TEXT NOT NULL DEFAULT 'AES-256-GCM',
  status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'rotating', 'retired')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  rotated_at  TIMESTAMPTZ
);

INSERT INTO secrets.encryption_keys (key_id, key_version, algorithm, status)
VALUES ('column-encryption-v1', 1, 'AES-256-GCM', 'active')
ON CONFLICT (key_id) DO NOTHING;

ALTER TABLE "user".users
  ALTER COLUMN date_of_birth TYPE TEXT USING date_of_birth::TEXT;

COMMENT ON TABLE secrets.encryption_keys IS
  'Application-layer DEK registry for encrypted PII fields (plan 10.13).';
